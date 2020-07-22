package gohalt

import (
	"context"
	"fmt"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/astaxie/beego"
	beegoctx "github.com/astaxie/beego/context"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	"github.com/kataras/iris/v12"
	"github.com/labstack/echo/v4"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"
	"github.com/revel/revel"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ip(req *http.Request) interface{} {
	first := func(ip string) string {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if ip := strings.TrimSpace(req.Header.Get("X-Real-Ip")); ip != "" {
		return first(ip)
	}
	if ip := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); ip != "" {
		return first(ip)
	}
	return first(req.RemoteAddr)
}

type GinWith func(*gin.Context) context.Context

func GinWithIP(gctx *gin.Context) context.Context {
	req := gctx.Request
	return WithKey(req.Context(), ip(req))
}

type GinOn func(*gin.Context, error)

func GinOnAbort(gctx *gin.Context, err error) {
	_ = gctx.AbortWithError(http.StatusTooManyRequests, err)
}

func NewMiddlewareGin(thr Throttler, with GinWith, on GinOn) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		r := NewRunnerSync(with(gctx), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				gctx.Header(key, val)
			}
			gctx.Next()
			return nil
		})
		if err := r.Result(); err != nil {
			on(gctx, err)
		}
	}
}

type StdWith func(*http.Request) context.Context

func StdWithIP(req *http.Request) context.Context {
	return WithKey(req.Context(), ip(req))
}

type StdOn func(http.ResponseWriter, error)

func StdOnAbort(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusTooManyRequests)
}

func NewMiddlewareStd(h http.Handler, thr Throttler, with StdWith, on StdOn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r := NewRunnerSync(with(req), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				w.Header().Add(key, val)
			}
			h.ServeHTTP(w, req)
			return nil
		})
		if err := r.Result(); err != nil {
			on(w, err)
		}
	})
}

type EchoWith func(echo.Context) context.Context

func EchoWithIP(ectx echo.Context) context.Context {
	req := ectx.Request()
	return WithKey(req.Context(), ip(req))
}

type EchoOn func(echo.Context, error) error

func EchoOnAbort(ectx echo.Context, err error) error {
	return ectx.String(http.StatusTooManyRequests, err.Error())
}

func NewMiddlewareEcho(thr Throttler, with EchoWith, on EchoOn) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ectx echo.Context) (err error) {
			r := NewRunnerSync(with(ectx), thr)
			r.Run(func(ctx context.Context) error {
				headers := NewMeta(ctx, thr).Headers()
				for key, val := range headers {
					ectx.Response().Header().Set(key, val)
				}
				err = next(ectx)
				return nil
			})
			if err := r.Result(); err != nil {
				return on(ectx, err)
			}
			return err
		}
	}
}

type BeegoWith func(*beegoctx.Context) context.Context

func BeegoWithIP(bctx *beegoctx.Context) context.Context {
	req := bctx.Request
	return WithKey(req.Context(), ip(req))
}

type BeegoOn func(*beegoctx.Context, error)

func BeegoOnAbort(bctx *beegoctx.Context, err error) {
	bctx.Abort(http.StatusTooManyRequests, err.Error())
}

func NewMiddlewareBeego(thr Throttler, with BeegoWith, on BeegoOn) beego.FilterFunc {
	return func(bctx *beegoctx.Context) {
		r := NewRunnerSync(with(bctx), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				bctx.Output.Header(key, val)
			}
			return nil
		})
		if err := r.Result(); err != nil {
			on(bctx, err)
		}
	}
}

type KitWith func(context.Context, interface{}) context.Context

func KitWithEmpty(ctx context.Context, req interface{}) context.Context {
	return ctx
}

type KitOn func(error) (interface{}, error)

func KitOnAbort(err error) (interface{}, error) {
	return nil, fmt.Errorf("%d: %w", http.StatusTooManyRequests, err)
}

func NewMiddlewareKit(thr Throttler, with KitWith, on KitOn) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			r := NewRunnerSync(with(ctx, req), thr)
			r.Run(func(ctx context.Context) error {
				resp, err = next(ctx, req)
				return nil
			})
			if err := r.Result(); err != nil {
				return on(err)
			}
			return resp, nil
		}
	}
}

type MuxWith StdWith

func MuxWithIP(req *http.Request) context.Context {
	return StdWithIP(req)
}

type MuxOn StdOn

func MuxOnAbort(w http.ResponseWriter, err error) {
	StdOnAbort(w, err)
}

func NewMiddlewareMux(h http.Handler, thr Throttler, with MuxWith, on MuxOn) http.Handler {
	return NewMiddlewareStd(h, thr, StdWith(with), StdOn(on))
}

type RouterWith StdWith

func RouterWithIP(req *http.Request) context.Context {
	return StdWithIP(req)
}

type RouterOn StdOn

func RouterOnAbort(w http.ResponseWriter, err error) {
	StdOnAbort(w, err)
}

func NewMiddlewareRouter(h http.Handler, thr Throttler, with MuxWith, on MuxOn) http.Handler {
	return NewMiddlewareStd(h, thr, StdWith(with), StdOn(on))
}

type RevealWith func(*revel.Controller) context.Context

func RevealWithIp(rc *revel.Controller) context.Context {
	req := rc.Request
	keys := req.Header.Server.GetKeys()
	stdreq := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: rc.ClientIP,
	}
	for _, key := range keys {
		stdreq.Header.Add(key, req.Header.Get(key))
	}
	return WithKey(req.Context(), ip(stdreq))
}

type RevealOn func(error) revel.Result

func RevealOnAbort(rc *revel.Controller, err error) revel.Result {
	result := rc.RenderError(err)
	rc.Response.Status = http.StatusTooManyRequests
	return result
}

func NewMiddlewareRevel(thr Throttler, with RevealWith, on RevealOn) revel.Filter {
	return func(rc *revel.Controller, chain []revel.Filter) {
		r := NewRunnerSync(with(rc), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				rc.Response.Out.Header().Add(key, val)
			}
			chain[0](rc, chain[1:])
			return nil
		})
		if err := r.Result(); err != nil {
			rc.Result = on(err)
		}
	}
}

type IrisWith func(iris.Context) context.Context

func IrisWithIP(ictx iris.Context) context.Context {
	req := ictx.Request()
	return WithKey(req.Context(), ip(req))
}

type IrisOn func(iris.Context, error)

func IrisOnAbort(ictx iris.Context, err error) {
	ictx.StatusCode(http.StatusTooManyRequests)
	_, _ = ictx.WriteString(err.Error())
}

func NewMiddlewareIris(thr Throttler, with IrisWith, on IrisOn) iris.Handler {
	return func(ictx iris.Context) {
		r := NewRunnerSync(with(ictx), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				ictx.Header(key, val)
			}
			ictx.Next()
			return nil
		})
		if err := r.Result(); err != nil {
			on(ictx, err)
		}
	}
}

type FastWith func(*fasthttp.RequestCtx) context.Context

func FastWithIPBackground(fctx *fasthttp.RequestCtx) context.Context {
	stdreq := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: fctx.RemoteIP().String(),
	}
	fctx.Request.Header.VisitAll(func(key []byte, val []byte) {
		stdreq.Header.Add(string(key), string(val))
	})
	return WithKey(context.Background(), ip(stdreq))
}

type FastOn func(*fasthttp.RequestCtx, error)

func FastOnAbort(fctx *fasthttp.RequestCtx, err error) {
	fctx.Error(err.Error(), fasthttp.StatusTooManyRequests)
}

func NewMiddlewareFast(h fasthttp.RequestHandler, thr Throttler, with FastWith, on FastOn) fasthttp.RequestHandler {
	return func(fctx *fasthttp.RequestCtx) {
		r := NewRunnerSync(with(fctx), thr)
		r.Run(func(ctx context.Context) error {
			headers := NewMeta(ctx, thr).Headers()
			for key, val := range headers {
				fctx.Response.Header.Add(key, val)
			}
			h(fctx)
			return nil
		})
		if err := r.Result(); err != nil {
			on(fctx, err)
		}
	}
}

type RoundTripperStdWith func(*http.Request) context.Context

func RoundTripperStdWithEmpty(req *http.Request) context.Context {
	return req.Context()
}

type RoundTripperStdOn func(error) error

func RoundTripperStdOnAbort(err error) error {
	return err
}

type rtstd struct {
	http.RoundTripper
	thr  Throttler
	with RoundTripperStdWith
	on   RoundTripperStdOn
}

func NewRoundTripperStd(rt http.RoundTripper, thr Throttler, with RoundTripperStdWith, on RoundTripperStdOn) rtstd {
	return rtstd{RoundTripper: rt, thr: thr, with: with, on: on}
}

func (rt rtstd) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	r := NewRunnerSync(rt.with(req), rt.thr)
	r.Run(func(ctx context.Context) error {
		headers := NewMeta(ctx, rt.thr).Headers()
		for key, val := range headers {
			req.Header.Add(key, val)
		}
		resp, err = rt.RoundTripper.RoundTrip(req)
		return nil
	})
	if err := r.Result(); err != nil {
		return nil, rt.on(err)
	}
	return resp, err
}

type RoundTripperFast interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type RoundTripperFastWith func(*fasthttp.Request) context.Context

func RoundTripperFastBackground(req *fasthttp.Request) context.Context {
	return context.Background()
}

type RoundTripperFastOn func(error) error

func RoundTripperFastOnAbort(err error) error {
	return err
}

type rtfast struct {
	RoundTripperFast
	thr  Throttler
	with RoundTripperFastWith
	on   RoundTripperFastOn
}

func NewRoundTripperFast(rt RoundTripperFast, thr Throttler, with RoundTripperFastWith, on RoundTripperFastOn) rtfast {
	return rtfast{RoundTripperFast: rt, thr: thr, with: with, on: on}
}

func (rt rtfast) Do(req *fasthttp.Request, resp *fasthttp.Response) (err error) {
	r := NewRunnerSync(rt.with(req), rt.thr)
	r.Run(func(ctx context.Context) error {
		headers := NewMeta(ctx, rt.thr).Headers()
		for key, val := range headers {
			req.Header.Add(key, val)
		}
		err = rt.RoundTripperFast.Do(req, resp)
		return nil
	})
	if err := r.Result(); err != nil {
		return rt.on(err)
	}
	return err
}

type RpcCodecWith func(*rpc.Request, *rpc.Response, interface{}) context.Context

func RpcCodecWithBackground(req *rpc.Request, resp *rpc.Response, msg interface{}) context.Context {
	return context.Background()
}

type RpcCodecOn func(error) error

func RpcCodecOnAbort(err error) error {
	return err
}

type rpcc struct {
	rpc.ClientCodec
	thr  Throttler
	with RpcCodecWith
	on   RpcCodecOn
}

func NewRpcClientCodec(cc rpc.ClientCodec, thr Throttler, with RpcCodecWith, on RpcCodecOn) rpcc {
	return rpcc{ClientCodec: cc, thr: thr, with: with, on: on}
}

func (cc rpcc) WriteRequest(req *rpc.Request, msg interface{}) (err error) {
	r := NewRunnerSync(cc.with(req, nil, msg), cc.thr)
	r.Run(func(ctx context.Context) error {
		err = cc.ClientCodec.WriteRequest(req, msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return cc.on(err)
	}
	return err
}

func (cc rpcc) ReadResponseHeader(resp *rpc.Response) (err error) {
	r := NewRunnerSync(cc.with(nil, resp, nil), cc.thr)
	r.Run(func(ctx context.Context) error {
		err = cc.ClientCodec.ReadResponseHeader(resp)
		return nil
	})
	if err := r.Result(); err != nil {
		return cc.on(err)
	}
	return err
}

type rpcs struct {
	rpc.ServerCodec
	thr  Throttler
	with RpcCodecWith
	on   RpcCodecOn
}

func NewRpcServerCodec(sc rpc.ServerCodec, thr Throttler, with RpcCodecWith, on RpcCodecOn) rpcs {
	return rpcs{ServerCodec: sc, thr: thr, with: with, on: on}
}

func (sc rpcs) ReadRequestHeader(req *rpc.Request) (err error) {
	r := NewRunnerSync(sc.with(req, nil, nil), sc.thr)
	r.Run(func(ctx context.Context) error {
		err = sc.ServerCodec.ReadRequestHeader(req)
		return nil
	})
	if err := r.Result(); err != nil {
		return sc.on(err)
	}
	return err
}

func (sc rpcs) WriteResponse(resp *rpc.Response, msg interface{}) (err error) {
	r := NewRunnerSync(sc.with(nil, resp, msg), sc.thr)
	r.Run(func(ctx context.Context) error {
		err = sc.ServerCodec.WriteResponse(resp, msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return sc.on(err)
	}
	return err
}

type GrpcStreamWith func(context.Context, interface{}) context.Context

func GrpcStreamWithEmpty(ctx context.Context, msg interface{}) context.Context {
	return ctx
}

type GrpcStreamOn func(error) error

func GrpcStreamAbort(err error) error {
	return err
}

type grpccs struct {
	grpc.ClientStream
	thr  Throttler
	with GrpcStreamWith
	on   GrpcStreamOn
}

func NewGrpClientStream(cs grpc.ClientStream, thr Throttler, with GrpcStreamWith, on GrpcStreamOn) grpccs {
	return grpccs{ClientStream: cs, thr: thr, with: with, on: on}
}

func (cs grpccs) SendMsg(msg interface{}) (err error) {
	r := NewRunnerSync(cs.with(cs.Context(), msg), cs.thr)
	r.Run(func(ctx context.Context) error {
		err = cs.ClientStream.SendMsg(msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return cs.on(err)
	}
	return err
}

func (cs grpccs) RecvMsg(msg interface{}) (err error) {
	r := NewRunnerSync(cs.with(cs.Context(), msg), cs.thr)
	r.Run(func(ctx context.Context) error {
		err = cs.ClientStream.RecvMsg(msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return cs.on(err)
	}
	return err
}

type grpcss struct {
	grpc.ServerStream
	thr  Throttler
	with GrpcStreamWith
	on   GrpcStreamOn
}

func NewGrpServerStream(ss grpc.ServerStream, thr Throttler, with GrpcStreamWith, on GrpcStreamOn) grpcss {
	return grpcss{ServerStream: ss, thr: thr, with: with, on: on}
}

func (ss grpcss) SetHeader(md metadata.MD) error {
	headers := NewMeta(ss.Context(), ss.thr).Headers()
	for key, val := range headers {
		md.Append(key, val)
	}
	return ss.ServerStream.SetHeader(md)
}

func (ss grpcss) SendMsg(msg interface{}) (err error) {
	r := NewRunnerSync(ss.with(ss.Context(), msg), ss.thr)
	r.Run(func(ctx context.Context) error {
		err = ss.ServerStream.SendMsg(msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return ss.on(err)
	}
	return err
}

func (ss grpcss) RecvMsg(msg interface{}) (err error) {
	r := NewRunnerSync(ss.with(ss.Context(), msg), ss.thr)
	r.Run(func(ctx context.Context) error {
		err = ss.ServerStream.RecvMsg(msg)
		return nil
	})
	if err := r.Result(); err != nil {
		return ss.on(err)
	}
	return err
}

type MicroClientWith func(context.Context, client.Request) context.Context

func MicroClientWithEmpty(ctx context.Context, req client.Request) context.Context {
	return ctx
}

type MicroServerWith func(context.Context, server.Request) context.Context

func MicroServerEmpty(ctx context.Context, req server.Request) context.Context {
	return ctx
}

type MicroOn func(error) error

func MicroOnAbort(err error) error {
	return err
}

type microcli struct {
	client.Client
	thr  Throttler
	with MicroClientWith
	on   MicroOn
}

func NewMicroClient(thr Throttler, with MicroClientWith, on MicroOn) client.Wrapper {
	return func(cli client.Client) client.Client {
		return microcli{Client: cli, thr: thr, with: with, on: on}
	}
}

func (cli microcli) Call(ctx context.Context, req client.Request, resp interface{}, opts ...client.CallOption) (err error) {
	r := NewRunnerSync(cli.with(ctx, req), cli.thr)
	r.Run(func(ctx context.Context) error {
		err = cli.Client.Call(ctx, req, resp, opts...)
		return nil
	})
	if err := r.Result(); err != nil {
		return cli.on(err)
	}
	return err
}

func NewMicroHandler(thr Throttler, with MicroServerWith, on MicroOn) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, resp interface{}) (err error) {
			r := NewRunnerSync(with(ctx, req), thr)
			r.Run(func(ctx context.Context) error {
				reqhead := req.Header()
				headers := NewMeta(ctx, thr).Headers()
				for key, val := range headers {
					reqhead[key] = val
				}
				err = h(ctx, req, resp)
				return nil
			})
			if err := r.Result(); err != nil {
				return on(err)
			}
			return err
		}
	}
}

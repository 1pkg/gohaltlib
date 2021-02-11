<p align="center">
    <img src="https://raw.githubusercontent.com/1pkg/gohalt/master/gopher.png" alt="gohalt"/>
</p>

# Gohaltlib 👮‍♀🔧: [Gohalt](https://github.com/1pkg/gohalt) 3rd Party Integrations library

[![lint](https://github.com/1pkg/gohaltlib/workflows/lint/badge.svg)](https://github.com/1pkg/gohaltlib/actions?query=workflow%3Alint+branch%3Amaster+)
[![test](https://github.com/1pkg/gohaltlib/workflows/test/badge.svg)](https://github.com/1pkg/gohaltlib/actions?query=workflow%3Atest+branch%3Amaster+)
[![report](https://goreportcard.com/badge/github.com/1pkg/gohaltlib?cache=1)](https://goreportcard.com/report/github.com/1pkg/gohaltlib)
[![version](https://img.shields.io/github/go-mod/go-version/1pkg/gohaltlib?cache=1)](https://github.com/1pkg/gohaltlib/blob/master/go.mod)
[![license](https://img.shields.io/github/license/1pkg/gohaltlib?cache=1)](LICENSE)
[![godoc](https://img.shields.io/badge/godoc-godoc-green?cache=1)](https://pkg.go.dev/github.com/1pkg/gohaltlib?tab=doc)

`go get -u github.com/1pkg/gohalt`
`go get -u github.com/1pkg/gohaltlib`

**Note:** in gohalt v0.3.0 all integrations were moved to separate gohaltlib repository to make base [gohalt](https://github.com/1pkg/gohalt) repository dependencies footprint small.

## Introduction

Gohalt is simple and convenient yet powerful and efficient throttling go library. Gohalt provides various throttlers and surronding tools to build throttling pipelines and rate limiters of any complexity adjusted to your specific needs. Gohalt provides an easy way to integrate throttling and rate limiting with your infrastructure through built in middlewares.

## Features

- [x] Blastly fast and efficient, Gohalt has minimal performance overhead, it was design with performance as the primary goal.
- [x] Flexible and powerful, Gohalt supports numbers of different throttling strategies and conditions that could be easily combined and customized to match your needs [link](https://github.com/1pkg/gohalt#Throttlers).
- [x] Easy to integrate, Gohalt provides separate package with numbers of built in middlewares for simple (couple lines of code) integrations with stdlib and other libraries, among which are: io, rpc/grpc, http, sql, gin, [etc](#Integrations).
- [x] Metrics awareness, Gohalt could use Prometheus metrics as a conditions for throttling.
- [x] Queueing and delayed processing, Gohalt supports throttling queueing which means you can easily save throttled query to rabbitmq/kafka stream to process it later.

- [ ] Durable storage, Gohalt has embedded k/v storage to provide thtottling persistence and durability.
- [ ] Meta awareness, Gohalt provides easy way to access inner throttlers state in form of meta that can be later exposed to logging, headers, etc. 

## Integrations

| Library | Adapter |
|---|---|
| gin | `func NewMiddlewareGin(thr Throttler, with GinWith, on GinOn) gin.HandlerFunc` |
| stdlib http handler | `func NewMiddlewareStd(h http.Handler, thr Throttler, with StdWith, on StdOn) http.Handler` |
| echo | `func NewMiddlewareEcho(thr Throttler, with EchoWith, on EchoOn) echo.MiddlewareFunc` |
| beego | `func NewMiddlewareBeego(thr Throttler, with BeegoWith, on BeegoOn) beego.FilterFunc` |
| kit | `func NewMiddlewareKit(thr Throttler, with KitWith, on KitOn) endpoint.Middleware` |
| mux | `func NewMiddlewareMux(h http.Handler, thr Throttler, with MuxWith, on MuxOn) http.Handler` |
| httprouter | `func NewMiddlewareRouter(h http.Handler, thr Throttler, with RouterWith, on RouterOn) http.Handler` |
| reveal | `func NewMiddlewareRevel(thr Throttler, with RevealWith, on RevealOn) revel.Filter` |
| iris | `func NewMiddlewareIris(thr Throttler, with IrisWith, on IrisOn) iris.Handler` |
| fasthttp | `func NewMiddlewareFast(h fasthttp.RequestHandler, thr Throttler, with FastWith, on FastOn) fasthttp.RequestHandler` |
| stdlib rt | `func NewRoundTripperStd(rt http.RoundTripper, thr Throttler, with RoundTripperStdWith, on RoundTripperStdOn) http.RoundTripper` |
| fasthttp rt | `func NewRoundTripperFast(rt RoundTripperFast, thr Throttler, with RoundTripperFastWith, on RoundTripperFastOn) RoundTripperFast` |
| stdlib rpc client coded | `func NewRPCClientCodec(cc rpc.ClientCodec, thr Throttler, with RPCCodecWith, on RPCCodecOn) rpc.ClientCodec` |
| stdlib rpc server coded | `func NewRPCServerCodec(sc rpc.ServerCodec, thr Throttler, with RPCCodecWith, on RPCCodecOn) rpc.ServerCodec` |
| grpc client stream | `func NewGRPCClientStream(cs grpc.ClientStream, thr Throttler, with GRPCStreamWith, on GRPCStreamOn) grpc.ClientStream` |
| grpc server stream | `func NewGrpServerStream(ss grpc.ServerStream, thr Throttler, with GRPCStreamWith, on GRPCStreamOn) grpc.ServerStream` |
| go-micro client | `func NewMicroClient(thr Throttler, with MicroClientWith, on MicroOn) client.Wrapper` |
| go-micro server | `func NewMicroHandler(thr Throttler, with MicroServerWith, on MicroOn) server.HandlerWrapper` |
| stdlib net conn | `func NewNetConn(conn net.Conn, thr Throttler, with NetConnWith, on NetConnOn, mode NetConnMode) net.Conn` |
| stdlib sql | `func NewSQLClient(cli SQLClient, thr Throttler, with SQLClientWith, on SQLClientOn) SQLClient` |
| stdlib io reader | `func NewReader(r io.Reader, thr Throttler, with RWWith, on RWOn) io.Reader` |
| stdlib io writer | `func NewWriter(w io.Writer, thr Throttler, with RWWith, on RWOn) io.Writer` |

## Licence

Gohaltlib is licensed under the MIT License.  
See [LICENSE](LICENSE) for the full license text.

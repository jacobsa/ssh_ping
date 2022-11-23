`ssh_ping` is a utility for measuring SSH session latency. It connects to a host
over SSH, then repeatedly sends data to be echoed back for five seconds and
measures statistics about the results.

Install and run it as follows:

```shell
> go install github.com/jacobsa/ssh_ping
> ssh_ping --host some.host.com
100 samples so far...
200 samples so far...
Collected 294 samples.

Min:      13.0 ms
p05:      13.6 ms
p50:      13.0 ms
p95:      21.1 ms
Max:      26.0 ms

Mean:     17.0 ms
Std. dev:  2.6 ms
```

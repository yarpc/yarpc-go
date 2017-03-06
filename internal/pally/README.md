# :two_women_holding_hands: pally

Pally makes [Prometheus][] and [tally][] pals. Rather than choosing one or the
other, take the best of both!

Compared to Prometheus, pally is fast: modeling metrics as integers instead of
floats allows pally to avoid expensive increment-and-compare loops.

Compared to tally, pally prioritizes introspection: like expvar and Prometheus,
all metrics can be inspected via a simple HTTP endpoint even when central
telemetry systems are down.

Pally grew out of the internal metrics libraries built by Uber's software
networking team. Its open-source incarnation is incubating in YARPC before
potentially migrating into an independent project.

Known to-dos:

- [ ] Histogram support
- [ ] Stopwatches (for convenient timing collection)
- [ ] Comparative benchmarks with Tally
- [ ] Comparative benchmarks with Prometheus

[Prometheus]: http://prometheus.io
[Tally]: https://github.com/uber-go/tally

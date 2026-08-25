# Design

Concrete interfaces, formats, and structures — the single owner of every field, command, and path. Other docs reference these, they don't restate them.

Boundary: no concrete name/format/rule → belongs in Architecture instead. A sequence of human steps → belongs in Runbooks instead. Split a doc once it crosses ~200 lines and register the split in the index below.

Template: [`_template.md`](_template.md). **Register new documents here — an unregistered doc doesn't count as existing.**

## Index

| Document | Topic |
| --- | --- |
| [fake-ip-routing.md](fake-ip-routing.md) | How DNS fake-ip, the route-layer reverse mapping, and `action: resolve` interact to decide whether a connection gets IPv4/IPv6 Happy Eyeballs racing or a single fixed address |

# For v1.0.0

- pull a design document out into docs
- resolve the config listen v port issue
- implement /meta/entity
- rework GenericDataFormat interface
  - allow preserving strings thru, rather than just []byte
  - consider splitting out the framing; either leave it always up to the wire
    protocol, or call it a "format default framing"?
  - add exported convenience types

# For after v1.0.0

- reporting support
- redis pub/sub pattern may provide better experience than monitor
- a collection agent based on watch and then later report

# Anytime

- more test coverage
- benchmarks
- more example integrations
  - metrics
  - runtime/trace
  - app logging

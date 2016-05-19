/*

Package gwr provides on demand operational data sources in Go.  A typical use
is adding in-depth debug tracing to your program that only turns on when there
are any active consumer(s).

Basics

A gwr data source is a named subset data that is Get-able and/or Watch-able.
The gwr library can then build reporting on top of Watch-able sources, or by
falling back to polling Get-able sources.

For example a request log source would be naturally Watch-able for future
requests as they come in.  An implementation could go further and add a
"last-10" buffer to also become Get-able.

Integrating

To bootstrap gwr and start its server listening on port 4040:

	gwr.Configure(&gwr.Config{ListenAddr: ":4040"})


GWR also adds a handler to the default http server; so if you already have a
default http server like:

	log.Fatal(http.ListenAndServe(":8080", nil))

Then gwr will already be accessible at "/gwr/..." on port 8080; you should
still call gwr.Configure:

	gwr.Configure(nil)

*/
package gwr

# net-hub

tcp-hub exposes a tcp port outside of the network without any NAT.

The tcp-hub client connects to:
* the application tcp server to expose
* the tcp-hub server

The tcp-hub server has two listeners:
* hub server waiting for a tcp-hub client
* external client to reach the application over the established tcp connection

Limitations:
* a single tcp connection can be forwarded

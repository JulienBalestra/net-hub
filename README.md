


Static IP / Server for clients
```
1.1.1.1: WAN IP
    NAT 9000 -> 10.0.0.1:9000
    NAT 9001 -> 10.0.0.1:9001
10.0.0.1: rpi4
```

socat command:
```bash
# 192.168.1.1: rpi4
socat -ddd TCP4-LISTEN:9001,reuseaddr,fork,ignoreeof,max-children=16 TCP4-LISTEN:9000,reuseaddr,fork,ignoreeof,max-children=16 
```
docs: http://www.dest-unreach.org/socat/doc/socat.html#ADDRESS_TCP_LISTEN

Isolated site / Bascom server to expose
```
192.168.1.1: 4G router
192.168.1.2: bascom
192.168.1.3: rpi4 
```

socat command:
```bash
# 192.168.1.3: rpi4
socat -ddd TCP4:1.1.1.1:9001,reuseaddr,end-close TCP4:192.168.1.2:9000,reuseaddr,end-close
```


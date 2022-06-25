# tun
This is an easy VPN server example by golang.
It support Tun interface only.

## 
```
+---------------+               +---------------+
|               |               |               |
|               |               |               |
+-------+-------+               +-------^-------+
        |                               |
+-------v-------+  works here   +-------+-------+
|Network Layer  <---------------+Network Layer  |
|Tun Interface  +--------------->Tun Interface  |
+-------+-------+               +-------^-------+
        |                               |
+-------v-------+               +-------+-------+
|Data Link Layer|               |Data Link Layer|
|Tap Interface  |               |Tap Interface  |
+-------+-------+               +-------^-------+
        |                               |
+-------v-------+               +-------+-------+
|Physical Layer |               |Physical Layer |
|               |               |               |
+-------+-------+               +-------^-------+
        |                               |
        +-------------------------------+
```

## Clients.
 - [XVPN](https://github.com/CrazyHulk/XVPN)：iOS 
 - [XVPN-Android](https://github.com/CrazyHulk/XVPN-Android)： Android
 - [macvpn](https://github.com/CrazyHulk/macvpn)：macOS 

## How to use?

Config your vpn.json first.

Enjoy yourself!

Must open ip_forwarding = 1

iptables -t nat -A POSTROUTING -j MASQUERADE

This is an easy VPN server example by golang.
It support Tun interface only.

# How to use?

Just run go run main.go, and you must config your vpn.json first.

Enjoy yourself!

Must open ip_forwarding = 1

iptables -t nat -A POSTROUTING -j MASQUERADE

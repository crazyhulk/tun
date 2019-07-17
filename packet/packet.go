package packet

type Packet []byte

func (p *Packet) Resize(length int) {
	if cap(*p) < length {
		old := *p
		*p = make(Packet, length, length)
		copy(*p, old)
	} else {
		*p = (*p)[:length]
	}
}

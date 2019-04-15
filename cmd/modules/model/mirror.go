package model

type MirrorsPoint struct {
	Name     string `json:"name"`
	Progress float64 `json:"progress"`
}

type MirrorsCdnPoint struct {
	MirrorId   string `json:"mirror_id"`
	NodeIpAddr string `json:"node_ip_addr"`
	Progress   float64 `json:"progress"`
}


package buffer

// todo remove when langchaingo supports

type Memories struct {
	Items []Memory `json:"memories"`
}

type Memory struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func (m *Memories) Add(m2 Memory) {
	m.Items = append(m.Items, m2)
}

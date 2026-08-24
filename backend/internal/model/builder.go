package model

func Builder(metric string) *LabelBuilder {
	return &LabelBuilder{ls: FromMap(metric, nil)}
}

type LabelBuilder struct {
	ls Labels
}

func (b *LabelBuilder) Set(k, v string) *LabelBuilder {
	found := false
	for i := range b.ls {
		if b.ls[i].Name == k {
			b.ls[i].Value = v
			found = true
			break
		}
	}
	if !found {
		b.ls = append(b.ls, Label{Name: k, Value: v})
	}
	b.ls = Normalize(b.ls)
	return b
}

func (b *LabelBuilder) Done() Labels { return b.ls }

package domain

import (
	"fmt"
	"strings"
)

type Placement struct {
	ID       ID     `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Version  int64  `json:"version"`
}

func NewPlacement(key, name string, capacity int) (Placement, error) {
	p := Placement{ID: NewID("plc"), Key: strings.TrimSpace(key), Name: strings.TrimSpace(name), Capacity: capacity, Version: 1}
	if p.Key == "" || p.Name == "" || capacity < 1 || capacity > 20 {
		return Placement{}, fmt.Errorf("placement key, name and capacity are invalid")
	}
	return p, nil
}

type Audience struct {
	ID         ID                `json:"id"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

func (a Audience) Matches(attributes map[string]string) bool {
	for key, expected := range a.Attributes {
		if attributes[key] != expected {
			return false
		}
	}
	return true
}

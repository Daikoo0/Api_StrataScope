package entity

type Role int

const (
	Owner Role = iota
	Editor
	Reader
)
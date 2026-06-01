package models

type Goal struct {
	ID        int
	Title     string
	Completed bool
	Claimed   bool
	Reward    int
}

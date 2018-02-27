package boson

import "time"

type Select []SelectExp

type Transform func(interface{}) (interface{}, error)

type SelectExp struct {
	Field     string
	Transform Transform
}

type From string

type GroupBy []int

type Downsampling int

type Between struct {
	Start time.Time
	End   time.Time
}

type Where []Predicate

type Match func(interface{}) (bool, error)

type Predicate struct {
	SelectExp SelectExp
	Match     Match
}

type Result []interface{}

type Wait func() error

type Boson interface {
	Query(Select, From, Where, GroupBy, Downsampling, Between) (<-chan []Result, Wait)
}

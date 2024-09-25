package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Members struct {
	Owner   string
	Editors []string
	Readers []string
}

type Config struct {
	Columns    map[string]bool `bson:"columns" json:"Columns"`
	IsInverted bool            `bson:"isInverted" json:"IsInverted"`
}

type Shared struct {
	Pass string `json:"pass"`
}

type Project struct {
	ID          primitive.ObjectID         `bson:"_id,omitempty"`
	ProjectInfo ProjectInfo                `bson:"projectinfo"`
	Data        []DataInfo                 `bson:"data"`
	Config      Config                     `bson:"config"`
	Fosil       map[string]Fosil           `bson:"fosil"`
	Facies      map[string][]FaciesSection `bson:"facies"`
	Shared      Shared                     `bson:"shared"`
}

type InfoProject struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ProjectInfo ProjectInfo        `bson:"projectinfo" json:"ProjectInfo"`
}

type ProjectInfo struct {
	Name         string  `bson:"name"`
	Owner        string  `bson:"owner"`
	Members      Members `bson:"members"`
	CreationDate string  `bson:"creationdate"`
	Description  string  `bson:"description"`
	Location     string  `bson:"location"`
	Lat          float64 `bson:"lat"`
	Long         float64 `bson:"long"`
	Visible      bool    `bson:"visible"`
}

type Data_project struct {
	Id_project primitive.ObjectID
	Data       []DataInfo                 `bson:"data"`
	Config     map[string]interface{}     `bson:"config"`
	Fosil      map[string]Fosil           `bson:"fosil"`
	Facies     map[string][]FaciesSection `bson:"facies"`
}

type DataInfo struct {
	Sistema                string         `json:"Sistema"`
	Edad                   string         `json:"Edad"`
	Formacion              string         `json:"Formacion"`
	Miembro                string         `json:"Miembro"`
	Espesor                string         `json:"Espesor"`
	Facie                  string         `json:"Facie"`
	AmbienteDepositacional string         `json:"AmbienteDepositacional"`
	Descripcion            string         `json:"Descripcion"`
	Litologia              LitologiaStruc `json:"Litologia"`
}

type LitologiaStruc struct {
	ColorFill   string `json:"ColorFill"`
	ColorStroke string `json:"ColorStroke"`
	File        string `json:"File"`
	Contact     string `json:"Contact"`
	//PrevContact string        `json:"PrevContact"`
	Zoom     int           `json:"Zoom"`
	Rotation int           `json:"Rotation"`
	Height   int           `json:"Height"`
	Tension  float32       `json:"Tension"`
	Circles  []CircleStruc `json:"Circles"`
}

type CircleStruc struct {
	X       float32 `json:"X"`
	Y       float32 `json:"Y"`
	Radius  float32 `json:"Radius"`
	Movable bool    `json:"Movable"`
	Name    string  `json:"Name"`
}

type FaciesSection struct {
	Y1 float32 `json:"y1"`
	Y2 float32 `json:"y2"`
}

type Fosil struct {
	Upper    int     `json:"upper"`
	Lower    int     `json:"lower"`
	FosilImg string  `json:"fosilImg"`
	X        float32 `json:"x"`
}

func NewFosil(upper int, lower int, fosilImg string, x float32) Fosil {
	return Fosil{
		Upper:    upper,
		Lower:    lower,
		FosilImg: fosilImg,
		X:        x,
	}
}

func NewCircle(point float32) CircleStruc {
	return CircleStruc{
		X:       0.5,
		Y:       point,
		Radius:  5,
		Movable: true,
		Name:    "none",
	}
}

func NewShape() DataInfo {
	return DataInfo{
		Sistema:                "",
		Edad:                   "",
		Formacion:              "",
		Miembro:                "",
		Espesor:                "",
		Facie:                  "",
		AmbienteDepositacional: "",
		Descripcion:            "",
		Litologia: LitologiaStruc{
			ColorFill:   "#ffffff",
			ColorStroke: "#000000",
			Zoom:        100,
			Rotation:    0,
			Tension:     0,
			File:        "Sin Pattern",
			Height:      100,
			Circles: []CircleStruc{
				{X: 0, Y: 0, Radius: 5, Movable: false},
				{X: 0.5, Y: 0, Radius: 5, Movable: true, Name: "none"},
				{X: 0.5, Y: 1, Radius: 5, Movable: true, Name: "none"},
				{X: 0, Y: 1, Radius: 5, Movable: false},
			},
			Contact: "111",
			//PrevContact: "111",
		},
	}
}

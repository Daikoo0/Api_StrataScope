package dtos

// Case editText
type EditText struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	RowIndex int    `json:"rowIndex"`
}

// Case add
type Add struct {
	RowIndex int     `json:"rowIndex"`
	Height   float32 `json:"height"`
}

// Case delete
type Delete struct {
	RowIndex int `json:"rowIndex"`
}

// Circle
type Circle struct {
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Radius  int     `json:"radius"`
	Movable bool    `json:"movable"`
}

// AddCircle
type AddCircle struct {
	RowIndex    int     `json:"rowIndex"`
	InsertIndex int     `json:"insertIndex"`
	Point       float32 `json:"point"`
}

// DeleteCircle
type DeleteCircle struct {
	RowIndex    int `json:"rowIndex"`
	DeleteIndex int `json:"deleteIndex"`
}

// EditCircle
type EditCircle struct {
	RowIndex  int     `json:"rowIndex"`
	EditIndex int     `json:"editIndex"`
	X         float32 `json:"x"`
	Name      string  `json:"name"`
}

// case fosil
type AddFosil struct {
	Upper    int     `json:"upper"`
	Lower    int     `json:"lower"`
	FosilImg string  `json:"fosilImg"`
	X        float32 `json:"x"`
}

type EditFosil struct {
	IdFosil  string  `json:"idFosil"`
	Upper    float32 `json:"upper"`
	Lower    float32 `json:"lower"`
	FosilImg string  `json:"fosilImg"`
	X        float32 `json:"x"`
}

type EditMuestra struct {
	IdMuestra   string  `json:"idMuestra"`
	Upper       float32 `json:"upper"`
	Lower       float32 `json:"lower"`
	MuestraText string  `json:"muestraText"`
	X           float32 `json:"x"`
}

type DeleteFosil struct {
	IdFosil string `json:"idFosil"`
}

type DeleteMuestra struct {
	IdMuestra string `json:"idMuestra"`
}

type Column struct {
	Column    string `json:"column"`
	IsVisible bool   `json:"isVisible"`
}

type IsInverted struct {
	IsInverted bool `json:"isInverted"`
}

type EditPolygon struct {
	RowIndex int         `json:"rowIndex"`
	Column   string      `json:"column"`
	Value    interface{} `json:"value"`
}

type UserEditingState struct {
	Section string `json:"section"`
	Name    string `json:"name"`
}

type Drop struct {
	ActiveId int `json:"activeId"`
	OverId   int `json:"overId"`
}

type Facie struct {
	Facie string `json:"facie"`
}

type AddFacieSection struct {
	Facie string  `json:"facie"`
	Y1    float32 `json:"y1"`
	Y2    float32 `json:"y2"`
	Index int     `json:"index"`
}

type DeleteFacieSection struct {
	Facie string `json:"facie"`
	Index int    `json:"index"`
}

type EditInfoProject struct {
	Name        string `json:"Name"`
	Location    string `json:"Location"`
	Visible     bool   `json:"Visible"`
	Description string `json:"Description"`
}

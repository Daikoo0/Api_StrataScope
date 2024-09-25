package api

import (
	"context"
	"encoding/json"
	"fmt"

	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"time"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/models"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lithammer/shortuuid/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ErrorMessage struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

type responseMessage struct {
	Message string `json:"message"`
}

type GeneralMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type ProjectResponse struct {
	Projects []models.InfoProject `json:"projects"`
}

type RoomData struct {
	ID              primitive.ObjectID
	ProjectInfo     models.ProjectInfo
	Data            []models.DataInfo
	Config          models.Config
	Fosil           map[string]models.Fosil
	Facies          map[string][]models.FaciesSection
	Shared          models.Shared
	Active          []*websocket.Conn
	SectionsEditing map[string]interface{}
	UserColors      map[string]string
	undoStack       []Action
	redoStack       []Action
}

var rooms = make(map[string]*RoomData)

var roomTimers = make(map[string]*time.Timer)
var roomActions = make(map[string]int)

var roomActionsThreshold = 10

func RemoveElement(a *API, ctx context.Context, roomID string, conn *websocket.Conn, name string, project *RoomData) {
	var index int = -1
	for i, c := range rooms[roomID].Active {
		if c == conn { // asumiendo que conn es comparable directamente
			index = i
			break
		}
	}

	if index != -1 {
		for key, value := range rooms[roomID].SectionsEditing {
			if value.(map[string]interface{})["name"] == name {
				delete(rooms[roomID].SectionsEditing, key)
				msgData := map[string]interface{}{
					"action": "deleteEditingUser",
					"value":  key,
					"name":   name,
				}

				sendSocketMessage(msgData, project, "deleteEditingUser")
			}
		}
		rooms[roomID].Active = append(rooms[roomID].Active[:index], rooms[roomID].Active[index+1:]...)

		if len(rooms[roomID].Active) == 0 {
			//guardar la sala
			err := a.repo.SaveRoom(context.Background(), models.Project{ID: project.ID, ProjectInfo: project.ProjectInfo, Data: project.Data, Config: project.Config, Fosil: project.Fosil, Facies: project.Facies, Shared: project.Shared})
			if err != nil {
				return
			}
			//	Eliminar la sala del mapa de salas si no hay usuarios conectados
			delete(rooms, roomID)
		}

		if _, exists := rooms[roomID]; exists {
			log.Println("La habitación", roomID, "no fue eliminada correctamente.")
		} else {
			log.Println("La habitación", roomID, "fue eliminada correctamente.")
		}
		log.Print("////////////////////////////////////////")
		log.Print(rooms)

	}

}

func (a *API) HandleWebSocket(c echo.Context) error {

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Println("Recuperado de un error/panic:", r)
	// 	}
	// }()

	ctx := c.Request().Context()
	roomID := c.Param("room")

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	auth := c.QueryParam("token")

	if auth == "" {
		errMessage := "Error: Unauthorized"
		conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
		conn.Close()
		return nil
	}

	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		errMessage := "Error: Unauthorized"
		conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
		conn.Close()
		return nil
	}

	user := claims["email"].(string)

	proyect := a.instanceRoom(ctx, roomID)
	if proyect == nil {
		errMessage := "Error: Room not found"
		conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
		conn.Close()
		return nil
	}

	permission := 2

	if proyect.ProjectInfo.Members.Owner == user {
		permission = 0
	} else if contains(proyect.ProjectInfo.Members.Editors, user) {
		permission = 1
	} else if contains(proyect.ProjectInfo.Members.Readers, user) {
		permission = 2
	} else if proyect.ProjectInfo.Visible {
		permission = 2
	} else {
		errMessage := ErrorMessage{
			Action:  "error",
			Message: "Access denied",
		}

		message, err := json.Marshal(errMessage)
		if err != nil {
			return err
		}

		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return err
		}

		conn.Close()
		return nil
	}

	proyect.Active = append(proyect.Active, conn)

	datos := proyect.Config.Columns

	orden := []string{"Sistema", "Edad", "Formacion", "Miembro", "Espesor", "Litologia", "Estructura fosil", "Facie", "AmbienteDepositacional", "Descripcion"}

	var claves []string

	for _, clave := range orden {
		if valor, existe := datos[clave]; existe && valor {
			claves = append(claves, clave)
		}
	}

	msgData := map[string]interface{}{
		"header":     claves,
		"isInverted": proyect.Config.IsInverted,
	}

	dataRoom := map[string]interface{}{
		"action":          "data",
		"projectInfo":     proyect.ProjectInfo,
		"data":            proyect.Data,
		"config":          msgData,
		"fosil":           proyect.Fosil,
		"facies":          proyect.Facies,
		"sectionsEditing": proyect.SectionsEditing,
	}

	databytes, err := json.Marshal(dataRoom)
	if err != nil {
		errMessage := "Error: cannot sent room config"
		conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
	}
	conn.WriteMessage(websocket.TextMessage, databytes)

	if err == nil {

		log.Println("Usuario conectado: ", user)
		log.Println("Permisos: ", permission)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if permission != 2 {
				var dataMap GeneralMessage
				err := json.Unmarshal([]byte(msg), &dataMap)

				if err != nil {
					log.Fatal(err)
				}

				if dataMap.Action != "editingUser" && dataMap.Action != "deleteEditingUser" && dataMap.Action != "columns" {
					go func() {
						// Bloquear acceso al mapa de timers para evitar condiciones de carrera
						a.saveMutex.Lock()
						defer a.saveMutex.Unlock()

						// Verificar si ya hay un timer activo para esta sala
						if timer, exists := roomTimers[roomID]; exists {

							if roomActions[roomID] >= roomActionsThreshold {

								// Función de guardado
								err = a.repo.SaveRoom(context.Background(), models.Project{ID: proyect.ID, ProjectInfo: proyect.ProjectInfo, Data: proyect.Data, Config: proyect.Config, Fosil: proyect.Fosil, Facies: proyect.Facies, Shared: proyect.Shared})
								if err != nil {
									log.Println("Error guardando la sala automáticamente: ", err)
								} else {
									log.Println("Sala guardada: ", roomID)
								}

								roomActions[roomID] = 0

								//salir de la funcion
								return
							}

							log.Println("Reiniciando el timer: ", roomID)
							log.Println("Acciones: ", roomActions[roomID])
							timer.Reset(5 * time.Second)
							roomActions[roomID]++

						} else {
							// Si no hay un timer, se crea uno
							log.Println("Creando un nuevo timer: ", roomID)
							timer := time.NewTimer(5 * time.Second)
							roomTimers[roomID] = timer

							go func() {
								<-timer.C

								// Función de guardado
								if rooms[roomID] != nil {
									err = a.repo.SaveRoom(context.Background(), models.Project{ID: proyect.ID, ProjectInfo: proyect.ProjectInfo, Data: proyect.Data, Config: proyect.Config, Fosil: proyect.Fosil, Facies: proyect.Facies, Shared: proyect.Shared})
									if err != nil {
										log.Println("Error guardando la sala automáticamente: ", err)
									} else {
										log.Println("Sala guardada: ", roomID)
									}
								}

								// Eliminar el timer del mapa una vez que la sala ha sido guardada
								a.saveMutex.Lock()
								delete(roomTimers, roomID)
								a.saveMutex.Unlock()
							}()
						}
					}()
				}
				switch dataMap.Action {

				case "undo":
					undo(proyect)

				case "redo":
					redo(proyect)

				case "tokenLink":

					tokenLink(conn, roomID, user, proyect)

				case "generateTokenLink":

					generateTokenLink(conn, roomID, user, proyect)

				case "editingUser":

					var editing dtos.UserEditingState
					err := json.Unmarshal(dataMap.Data, &editing)
					if err != nil {
						log.Println("Error al deserializar: ", err)
					}

					//var name = claims["name"].(string)
					section := editing.Section
					roomData := rooms[roomID]

					color, exists := roomData.UserColors[user]
					if !exists {
						// Generar un color único para el usuario y guardarlo en el mapa
						color = generateRandomColor()
						roomData.UserColors[user] = color
					}

					if roomData.SectionsEditing == nil {
						// Si no está inicializado, inicialízalo
						roomData.SectionsEditing = make(map[string]interface{})
					}

					msgData := map[string]interface{}{
						"action":   "editingUser",
						"userName": user,
						"color":    color,
						"value":    section,
					}

					roomData.SectionsEditing[section] = map[string]interface{}{
						"name":  user,
						"color": color,
						//"section" : section,
					}
					sendSocketMessage(msgData, proyect, "editingUser")

				case "deleteEditingUser":
					var editing dtos.UserEditingState
					err := json.Unmarshal(dataMap.Data, &editing)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}
					section := editing.Section
					roomData := rooms[roomID]
					name := editing.Name

					if roomData.SectionsEditing != nil {
						if _, ok := roomData.SectionsEditing[section]; ok {
							delete(roomData.SectionsEditing, section)

							msgData := map[string]interface{}{
								"action": "deleteEditingUser",
								"value":  section,
								"name":   name,
							}

							sendSocketMessage(msgData, proyect, "deleteEditingUser")
						} else {
							log.Println("El elemento a eliminar no existe")
						}
					}

				case "añadir":

					var addData dtos.Add
					err := json.Unmarshal(dataMap.Data, &addData)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					performAction(proyect,
						Action{
							Execute: func() {
								añadir(proyect, addData, models.NewShape())
							},
							Undo: func() {
								deleteRow(proyect, dtos.Delete{RowIndex: addData.RowIndex})
							},
						},
					)

				case "addCircle":

					var addCircleData dtos.AddCircle
					err := json.Unmarshal(dataMap.Data, &addCircleData)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					performAction(proyect,
						Action{
							Execute: func() {
								addCircle(proyect, addCircleData, models.NewCircle(addCircleData.Point))
							},
							Undo: func() {
								deleteCircle(proyect, dtos.DeleteCircle{RowIndex: addCircleData.RowIndex, DeleteIndex: addCircleData.InsertIndex})
							},
						},
					)

				case "addFosil":

					var fosil models.Fosil
					err := json.Unmarshal(dataMap.Data, &fosil)
					if err != nil {
						log.Println("Error", err)
						break
					}

					id := shortuuid.New()

					performAction(proyect,
						Action{
							Execute: func() {
								addFosil(proyect, id, fosil)
							},
							Undo: func() {
								deleteFosil(proyect, dtos.DeleteFosil{IdFosil: id})
							},
						},
					)

				case "addFacie":

					var facie dtos.Facie
					err := json.Unmarshal(dataMap.Data, &facie)
					if err != nil {
						log.Println("Error", err)
						break
					}

					performAction(proyect,
						Action{
							Execute: func() {
								addFacie(proyect, facie, nil)
							},
							Undo: func() {
								deleteFacie(proyect, facie)
							},
						},
					)

				case "addFacieSection":

					var f dtos.AddFacieSection
					err := json.Unmarshal(dataMap.Data, &f)
					if err != nil {
						log.Println("Error", err)
						break
					}

					previousIndex := len(proyect.Facies[f.Facie])

					performAction(proyect,
						Action{
							Execute: func() {
								addFacieSection(proyect, f, models.FaciesSection{Y1: f.Y1, Y2: f.Y2})
							},
							Undo: func() {
								deleteFacieSection(proyect, dtos.DeleteFacieSection{Facie: f.Facie, Index: previousIndex})
							},
						},
					)

				case "editCircle":

					var editCircles dtos.EditCircle
					err := json.Unmarshal(dataMap.Data, &editCircles)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					oldx := proyect.Data[editCircles.RowIndex].Litologia.Circles[editCircles.EditIndex].X
					oldname := proyect.Data[editCircles.RowIndex].Litologia.Circles[editCircles.EditIndex].Name

					performAction(proyect,
						Action{
							Execute: func() {
								editCircle(proyect, editCircles)
							},
							Undo: func() {
								editCircle(proyect, dtos.EditCircle{RowIndex: editCircles.RowIndex, EditIndex: editCircles.EditIndex, X: oldx, Name: oldname})
							},
						},
					)

				case "editText":

					var editTextData dtos.EditText
					err := json.Unmarshal(dataMap.Data, &editTextData)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					oldvalue := GetFieldString(proyect.Data[editTextData.RowIndex], editTextData.Key)
					textData := editTextData
					textData.Value = oldvalue

					performAction(proyect,
						Action{
							Execute: func() {
								editText(proyect, editTextData)
							},
							Undo: func() {
								editText(proyect, textData)
							},
						},
					)

				case "editPolygon":

					var polygon dtos.EditPolygon
					err := json.Unmarshal(dataMap.Data, &polygon)
					if err != nil {
						log.Println("Error deserializando el polygon:", err)
						break
					}

					oldvalue := GetFieldString(proyect.Data[polygon.RowIndex].Litologia, polygon.Column)
					editpolygon := polygon
					editpolygon.Value = oldvalue

					performAction(proyect,
						Action{
							Execute: func() {
								editPolygon(proyect, polygon)
							},
							Undo: func() {
								editPolygon(proyect, editpolygon)
							},
						},
					)

				case "editFosil":

					var fosil dtos.EditFosil
					err := json.Unmarshal(dataMap.Data, &fosil)
					if err != nil {
						log.Println("Error deserializando fósil:", err)
						break
					}

					oldFosil := proyect.Fosil[fosil.IdFosil]

					performAction(proyect,
						Action{
							Execute: func() {
								editFosil(proyect, fosil.IdFosil, models.NewFosil(fosil.Upper, fosil.Lower, fosil.FosilImg, fosil.X))
							},
							Undo: func() {
								editFosil(proyect, fosil.IdFosil, oldFosil)
							},
						},
					)

				case "delete":

					var deleteData dtos.Delete
					err := json.Unmarshal(dataMap.Data, &deleteData)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					copia := proyect.Data[deleteData.RowIndex]

					performAction(proyect,
						Action{
							Execute: func() {
								deleteRow(proyect, deleteData)
							},
							Undo: func() {
								añadir(proyect, dtos.Add{RowIndex: deleteData.RowIndex}, copia)
							},
						},
					)

				case "deleteCircle":

					var delCircle dtos.DeleteCircle
					err := json.Unmarshal(dataMap.Data, &delCircle)
					if err != nil {
						log.Println("Error al deserializar: ", err)
						break
					}

					oldcircle := proyect.Data[delCircle.RowIndex].Litologia.Circles[delCircle.DeleteIndex]

					performAction(proyect,
						Action{
							Execute: func() {
								deleteCircle(proyect, delCircle)
							},
							Undo: func() {
								addCircle(proyect, dtos.AddCircle{RowIndex: delCircle.RowIndex, InsertIndex: delCircle.DeleteIndex}, oldcircle)
							},
						},
					)

				case "deleteFosil":

					var fosilID dtos.DeleteFosil
					err := json.Unmarshal(dataMap.Data, &fosilID)
					if err != nil {
						log.Println("Error deserializando fósil:", err)
						break
					}

					fosil := proyect.Fosil[fosilID.IdFosil]

					performAction(proyect,
						Action{
							Execute: func() {
								deleteFosil(proyect, fosilID)
							},
							Undo: func() {
								addFosil(proyect, fosilID.IdFosil, fosil)
							},
						},
					)

				case "deleteFacie":

					var facie dtos.Facie
					err := json.Unmarshal(dataMap.Data, &facie)
					if err != nil {
						log.Println("Error", err)
						break
					}

					oldfacie := proyect.Facies[facie.Facie]

					performAction(proyect,
						Action{
							Execute: func() {
								deleteFacie(proyect, facie)
							},
							Undo: func() {
								addFacie(proyect, facie, oldfacie)
							},
						},
					)

				case "deleteFacieSection":

					var f dtos.DeleteFacieSection
					err := json.Unmarshal(dataMap.Data, &f)
					if err != nil {
						log.Println("Error", err)
						break
					}

					var removedSection models.FaciesSection
					if sections, exists := proyect.Facies[f.Facie]; exists && f.Index >= 0 && f.Index < len(sections) {
						removedSection = sections[f.Index]
					}

					performAction(proyect,
						Action{
							Execute: func() {
								deleteFacieSection(proyect, f)
							},
							Undo: func() {
								addFacieSection(proyect, dtos.AddFacieSection{Facie: f.Facie, Index: f.Index}, removedSection)
							},
						},
					)

				case "isInverted":

					isInverted(proyect, dataMap)

				case "save":

					a.save(proyect)

				case "columns":
					var column dtos.Column
					err := json.Unmarshal(dataMap.Data, &column)
					if err != nil {
						log.Println("Error deserializando columna:", err)
						break
					}
					// datos := rooms[roomID].Config["columns"].(map[string]interface{})
					datos := rooms[roomID].Config.Columns
					datos[column.Column] = column.IsVisible

					// Crear un slice para almacenar las columnas ordenadas
					//orderedColumns := make([]interface{}, len(orden))
					var orderedVisibleColumns []string

					// Llenar el slice con los datos de las columnas en el orden correcto
					for _, colName := range orden {
						if isVisible, ok := datos[colName]; ok && isVisible {
							// Si la columna es visible (IsVisible == true), agregar su nombre al slice.
							orderedVisibleColumns = append(orderedVisibleColumns, colName)
						}
					}

					msgData := map[string]interface{}{
						"action":  "columns",
						"columns": orderedVisibleColumns,
					}

					sendSocketMessage(msgData, proyect, "columns")

				}

			} else {
				errMessage := "Error: Don't have permission to edit this document"
				conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
			}
		}
	}

	conn.Close()
	log.Print("Me pitie la sala") //Mandar aviso para que no trate de reconectar
	RemoveElement(a, ctx, roomID, conn, claims["email"].(string), proyect)
	return nil
}

func sendSocketMessage(msgData map[string]interface{}, proyect *RoomData, action string) {

	jsonMsg, err := json.Marshal(msgData)
	if err != nil {
		log.Fatal("Error al serializar mensaje:", err)
	}

	for _, client := range proyect.Active {
		err = client.WriteMessage(websocket.TextMessage, jsonMsg)
		if err != nil {
			log.Println("Error al enviar mensaje:", err)
			log.Println("action: ", action)
		}
	}

}

func (a *API) instanceRoom(ctx context.Context, roomID string) *RoomData {

	existingRoom, exists := rooms[roomID]
	if exists {
		return existingRoom
	}

	room, err := a.serv.GetRoom(ctx, roomID)
	if err != nil {
		return nil
	}

	newRoom := &RoomData{
		ID:              room.ID,
		ProjectInfo:     room.ProjectInfo,
		Data:            room.Data,
		Config:          room.Config,
		Fosil:           room.Fosil,
		Facies:          room.Facies,
		Shared:          room.Shared,
		Active:          make([]*websocket.Conn, 0),
		SectionsEditing: make(map[string]interface{}),
		UserColors:      make(map[string]string),
		undoStack:       make([]Action, 0),
		redoStack:       make([]Action, 0),
	}

	rooms[roomID] = newRoom
	return newRoom
}

func (a *API) HandleGetActiveProject(c echo.Context) error {

	var keys []string

	for key := range rooms {
		keys = append(keys, key)
	}

	return c.JSON(http.StatusOK, keys)
}

// Generar un color aleatorio en formato hexadecimal
func generateRandomColor() string {
	color := make([]byte, 3)
	_, err := rand.Read(color)
	if err != nil {
		panic(err)
	}
	return "#" + hex.EncodeToString(color)
}

// Generar una contraseña aleatoria de n bytes
func generateRandomPass(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("error generando pass aleatorio: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func añadir(project *RoomData, addData dtos.Add, newShape models.DataInfo) {
	rowIndex := addData.RowIndex
	height := addData.Height

	// Actualiza la altura si se proporciona
	if height != 0 {
		newShape.Litologia.Height = height
	}

	msgData := map[string]interface{}{
		"action": "añadir",
		"value":  newShape,
	}

	// Agrega el nuevo shape en la posición correspondiente
	if rowIndex == -1 { // Agrega al final
		project.Data = append(project.Data, newShape)
		msgData["action"] = "añadirEnd"
	} else { // Agrega en la posición indicada
		project.Data = append(project.Data[:rowIndex], append([]models.DataInfo{newShape}, project.Data[rowIndex:]...)...)
		msgData["rowIndex"] = rowIndex
	}

	// Envía el mensaje al socket
	sendSocketMessage(msgData, project, msgData["action"].(string))
}

func deleteRow(project *RoomData, deleteData dtos.Delete) {

	rowIndex := deleteData.RowIndex

	if rowIndex == -1 {
		rowIndex = len(project.Data) - 1
	}

	if rowIndex < 0 || rowIndex >= len(project.Data) {
		log.Println("Índice fuera de los límites")
		return
	}

	// Eliminar la fila especificada
	project.Data = append(project.Data[:rowIndex], project.Data[rowIndex+1:]...)

	// Preparar y enviar el mensaje de eliminación
	msgData := map[string]interface{}{
		"action":   "delete",
		"rowIndex": rowIndex,
	}
	sendSocketMessage(msgData, project, "delete")
}

func editText(project *RoomData, editTextData dtos.EditText) {

	key := editTextData.Key
	value := editTextData.Value
	rowIndex := editTextData.RowIndex

	roomData := &project.Data[rowIndex]

	UpdateFieldAll(roomData, key, value)

	msgData := map[string]interface{}{
		"action":   "editText",
		"key":      key,
		"value":    value,
		"rowIndex": rowIndex,
	}

	sendSocketMessage(msgData, project, "editText")

}

func editPolygon(project *RoomData, polygon dtos.EditPolygon) {

	rowIndex := polygon.RowIndex
	column := polygon.Column
	value := polygon.Value

	roomData := &project.Data[rowIndex].Litologia

	// Actualiza el campo correspondiente en Litologia
	UpdateFieldLit(roomData, column, value)

	msgData := map[string]interface{}{
		"action":   "editPolygon",
		"rowIndex": rowIndex,
		"key":      column,
		"value":    value,
	}

	sendSocketMessage(msgData, project, "editPolygon")
}

func addCircle(project *RoomData, addCircleData dtos.AddCircle, newCircle models.CircleStruc) {

	rowIndex := addCircleData.RowIndex
	insertIndex := addCircleData.InsertIndex

	roomData := &project.Data[rowIndex].Litologia.Circles

	*roomData = append((*roomData)[:insertIndex], append([]models.CircleStruc{newCircle}, (*roomData)[insertIndex:]...)...)

	// Enviar informacion a los clientes
	msgData := map[string]interface{}{
		"action":   "addCircle",
		"rowIndex": rowIndex,
		"value":    roomData,
	}

	sendSocketMessage(msgData, project, "addCircle")

}

func deleteCircle(project *RoomData, deleteCircleData dtos.DeleteCircle) {

	rowIndex := deleteCircleData.RowIndex
	deleteIndex := deleteCircleData.DeleteIndex

	roomData := &project.Data[rowIndex].Litologia.Circles

	*roomData = append((*roomData)[:deleteIndex], (*roomData)[deleteIndex+1:]...)

	msgData := map[string]interface{}{
		"action":   "addCircle",
		"rowIndex": rowIndex,
		"value":    roomData,
	}

	sendSocketMessage(msgData, project, "deleteCircle")

}

func editCircle(project *RoomData, editCircleData dtos.EditCircle) {

	rowIndex := editCircleData.RowIndex
	editIndex := editCircleData.EditIndex
	x := editCircleData.X
	name := editCircleData.Name

	roomData := &project.Data[rowIndex].Litologia.Circles

	(*roomData)[editIndex].X = x
	(*roomData)[editIndex].Name = name

	msgData := map[string]interface{}{
		"action":   "addCircle",
		"rowIndex": rowIndex,
		"value":    roomData,
	}

	sendSocketMessage(msgData, project, "editCircle")

}

func addFosil(project *RoomData, id string, newFosil models.Fosil) {

	roomData := &project.Fosil
	(*roomData)[id] = newFosil

	msgData := map[string]interface{}{
		"action":  "addFosil",
		"idFosil": id,
		"value":   newFosil,
	}

	sendSocketMessage(msgData, project, "addFosil")

}

func deleteFosil(project *RoomData, fosilID dtos.DeleteFosil) {

	id := fosilID.IdFosil

	roomData := &project.Fosil
	delete(*roomData, id)

	msgData := map[string]interface{}{
		"action":  "deleteFosil",
		"idFosil": id,
	}

	sendSocketMessage(msgData, project, "deleteFosil")

}

func editFosil(project *RoomData, id string, newFosil models.Fosil) {

	roomData := &project.Fosil
	(*roomData)[id] = newFosil

	msgData := map[string]interface{}{
		"action":  "editFosil",
		"idFosil": id,
		"value":   newFosil,
	}

	sendSocketMessage(msgData, project, "editFosil")

}

func addFacie(project *RoomData, facie dtos.Facie, sections []models.FaciesSection) {

	name := facie.Facie

	if project.Facies == nil {
		project.Facies = make(map[string][]models.FaciesSection)
	}

	if sections != nil {
		project.Facies[name] = sections
	} else {
		project.Facies[name] = []models.FaciesSection{}
	}

	msgData := map[string]interface{}{
		"action":   "addFacie",
		"facie":    name,
		"sections": sections,
	}

	sendSocketMessage(msgData, project, "addFacie")
}

func deleteFacie(project *RoomData, facie dtos.Facie) {

	id := facie.Facie

	roomData := &project.Facies
	delete(*roomData, id)

	msgData := map[string]interface{}{
		"action": "deleteFacie",
		"facie":  id,
	}

	sendSocketMessage(msgData, project, "deleteFacie")

}

func isInverted(project *RoomData, dataMap GeneralMessage) {

	var isInverted dtos.IsInverted
	err := json.Unmarshal(dataMap.Data, &isInverted)
	if err != nil {
		log.Println("Error deserializando columna:", err)
		return
	}

	project.Config.IsInverted = isInverted.IsInverted

	msgData := map[string]interface{}{
		"action":     "isInverted",
		"isInverted": isInverted.IsInverted,
	}

	sendSocketMessage(msgData, project, "isInverted")

}

func (a *API) save(project *RoomData) {

	err := a.repo.SaveRoom(context.Background(), models.Project{ID: project.ID, ProjectInfo: project.ProjectInfo, Data: project.Data, Config: project.Config, Fosil: project.Fosil, Facies: project.Facies, Shared: project.Shared})
	if err != nil {
		log.Println("No se guardo la data")
	}

}

func addFacieSection(project *RoomData, f dtos.AddFacieSection, section models.FaciesSection) {

	name := f.Facie

	// Restaurar la sección en la posición original si es necesario
	if f.Index >= 0 && f.Index <= len(project.Facies[name]) {
		project.Facies[name] = append(project.Facies[name][:f.Index], append([]models.FaciesSection{section}, project.Facies[name][f.Index:]...)...)
	} else {
		// Añadir la sección al final si no se proporciona una posición válida
		project.Facies[name] = append(project.Facies[name], section)
	}

	msgData := map[string]interface{}{
		"action": "addFacieSection",
		"facie":  name,
		"y1":     section.Y1,
		"y2":     section.Y2,
	}

	sendSocketMessage(msgData, project, "addFacieSection")
}

func deleteFacieSection(project *RoomData, f dtos.DeleteFacieSection) {

	name := f.Facie
	index := f.Index

	innerMap := project.Facies[name]

	if index >= 0 && index < len(innerMap) {
		innerMap = append(innerMap[:index], innerMap[index+1:]...)
	}

	project.Facies[name] = innerMap

	msgData := map[string]interface{}{
		"action": "deleteFacieSection",
		"facie":  name,
		"index":  index,
	}

	sendSocketMessage(msgData, project, "deleteFacieSection")
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func generateTokenLink(conn *websocket.Conn, roomID string, user string, proyect *RoomData) {

	storedpass, err := generateRandomPass(8)
	if err != nil {
		log.Println("Error generando contraseña aleatoria: ", err)
		return
	}

	editorToken, err := encryption.InviteToken(roomID, "editors", storedpass)
	if err != nil {
		log.Println("Error generando token de editor: ", err)
		return
	}

	readerToken, err := encryption.InviteToken(roomID, "readers", storedpass)
	if err != nil {
		log.Println("Error generando token de lector: ", err)
		return
	}

	proyect.Shared.Pass = storedpass

	msgData := map[string]interface{}{
		"action": "tokenLink",
		"editor": editorToken,
		"reader": readerToken,
	}

	shareproyect, err := json.Marshal(msgData)
	if err != nil {
		log.Println("Error al serializar mensaje:", err)
	}

	if user == proyect.ProjectInfo.Members.Owner {
		conn.WriteMessage(websocket.TextMessage, shareproyect)
	}

}

func tokenLink(conn *websocket.Conn, roomID string, user string, proyect *RoomData) {
	if user == proyect.ProjectInfo.Members.Owner {

		storedpass := proyect.Shared.Pass
		if storedpass == "" {
			return
		}

		editorToken, err := encryption.InviteToken(roomID, "editors", storedpass)
		if err != nil {
			log.Println("Error generando token de editor: ", err)
			return
		}

		readerToken, err := encryption.InviteToken(roomID, "readers", storedpass)
		if err != nil {
			log.Println("Error generando token de lector: ", err)
			return
		}

		msgData := map[string]interface{}{
			"action": "tokenLink",
			"editor": editorToken,
			"reader": readerToken,
		}

		shareproyect, err := json.Marshal(msgData)
		if err != nil {
			log.Println("Error al serializar mensaje:", err)
		}

		conn.WriteMessage(websocket.TextMessage, shareproyect)
	}

}

func (a *API) ValidateInvitation(c echo.Context) error {

	ctx := c.Request().Context()
	log.Println("Validando invitación")

	// Revisar Token de autenticación
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}
	claimsAuth, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	// Revisar Token de Invitación
	var requestBody struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	claims, err := encryption.ParseInviteToken(requestBody.Token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
	}

	email := claimsAuth["email"].(string)
	storedPass := claims.Pass

	// Verificar si la sala está en memoria
	existingRoom, exists := rooms[claims.RoomID]
	var members *models.Members
	var pass string
	if exists {
		members = &existingRoom.ProjectInfo.Members
		pass = existingRoom.Shared.Pass
	} else {
		var err error
		members, pass, err = a.repo.GetMembersAndPass(ctx, claims.RoomID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, responseMessage{Message: err.Error()})
		}
	}

	if pass != storedPass {
		log.Println("Contraseña incorrecta")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
	}

	response := map[string]interface{}{
		"status": "valid",
		"roomID": claims.RoomID,
		"role":   claims.Role,
	}

	// Usuario ya es miembro
	if members.Owner == email || contains(members.Editors, email) || contains(members.Readers, email) {
		return c.JSON(http.StatusOK, response)
	}

	// Si el usuario no es miembro, añadirlo
	if exists {
		switch claims.Role {
		case "editors":
			existingRoom.ProjectInfo.Members.Editors = append(existingRoom.ProjectInfo.Members.Editors, email)
		case "readers":
			existingRoom.ProjectInfo.Members.Readers = append(existingRoom.ProjectInfo.Members.Readers, email)
		}
	} else {
		err := a.repo.AddUserToProject(context.Background(), email, claims.Role, claims.RoomID)
		if err != nil {
			log.Print("Error añadiendo usuario a la sala: ")
			return c.JSON(http.StatusInternalServerError, responseMessage{Message: err.Error()})
		}
	}

	return c.JSON(http.StatusOK, response)
}

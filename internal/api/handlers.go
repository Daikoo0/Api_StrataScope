package api

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"

	"encoding/hex"
	"log"
	"math/rand"
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

var colors = []string{
	"#FF5733", // Rojo
	"#33FF57", // Verde
	"#3357FF", // Azul
	"#F0E68C", // Khaki
	"#FF33A6", // Rosa
	"#33FFF8", // Cyan
	"#FF8333", // Naranja
	"#B3FF33", // Lima
	"#C33FFF", // Púrpura
}

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
	Projects    []models.InfoProject `json:"projects"`
	CurrentPage int                  `json:"currentPage"`
	TotalPages  int                  `json:"totalPages"`
}

type UserConnection struct {
	Email   string
	Conn    *websocket.Conn
	Editing string
	Color   string
}

type RoomData struct {
	mu             sync.Mutex
	ID             primitive.ObjectID
	ProjectInfo    models.ProjectInfo
	Data           []models.DataInfo
	Config         models.Config
	Fosil          map[string]models.Fosil
	Facies         map[string][]models.FaciesSection
	Muestras       map[string]models.Muestra
	Shared         models.Shared
	Active         map[string]*UserConnection
	undoStack      []Action
	redoStack      []Action
	saveTimer      *time.Timer
	actionsCounter int
}

var rooms sync.Map

// var rooms = make(map[string]*RoomData)

var roomActionsThreshold = 30

func RemoveElement(a *API, roomID string, userID string, project *RoomData) {

	err := project.DisconnectUser(userID)
	if err != nil {
		log.Println("Disconnect: ", err)
		return
	}

	// Si no hay más usuarios conectados, guardar y eliminar la sala
	if len(project.Active) == 0 {
		err := a.repo.SaveRoom(context.Background(), models.Project{
			ID:          project.ID,
			ProjectInfo: project.ProjectInfo,
			Data:        project.Data,
			Config:      project.Config,
			Fosil:       project.Fosil,
			Facies:      project.Facies,
			Muestras:    project.Muestras,
			Shared:      project.Shared,
		})
		if err != nil {
			log.Println("Error al guardar la sala:", err)
			return
		}

		log.Println("Project saved: ", project.ID)

		// delete(rooms, roomID)
		//rooms.Delete(roomID)
		// log.Println("Deleted room: ", roomID)
	}

	// log.Print(rooms)
}

func (a *API) HandleWebSocket(c echo.Context) error {

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
		conn.WriteJSON(ErrorMessage{Action: "error", Message: "Access denied"})
		conn.Close()
		return nil
	}

	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		conn.WriteJSON(ErrorMessage{Action: "error", Message: err.Error()})
		conn.Close()
		return nil
	}

	user := claims["email"].(string)
	userID := shortuuid.New()

	proyect := a.instanceRoom(ctx, roomID)
	if proyect == nil {
		conn.WriteJSON(ErrorMessage{Action: "error", Message: "Room not found"})
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
		conn.WriteJSON(ErrorMessage{Action: "error", Message: "Access denied"})
		conn.Close()
		return nil
	}

	// orden := []string{"Sistema", "Edad", "Formacion", "Miembro", "Espesor", "Litologia", "Estructura fosil", "Facie", "AmbienteDepositacional", "Descripcion"}

	dataRoom := proyect.DataProject()

	if err = conn.WriteJSON(dataRoom); err == nil {

		proyect.AddUser(conn, user, userID)
		sendSocketMessage(map[string]interface{}{"action": "userConnected", "id": userID, "mail": user, "color": proyect.Active[userID].Color}, proyect, "userConnected")
		// log de usuario conectado y permisos
		log.Println("\033[36m User connected: ", user, " permission: ", permission, "\033[0m")

		// Configuramos el handler para manejar el "pong"
		// conn.SetPongHandler(func(appData string) error {
		// 	log.Println("Pong recibido de: ", user)
		// 	return nil
		// })

		go func() {
			//Inicia el timer
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					conn.Close()
					if len(proyect.Active) == 0 {
						_, exists := rooms.Load(roomID)
						if exists {
							rooms.Delete(roomID)
							log.Println("Deleted room: ", roomID)
						}
					}
					return
				}
			}

		}()

		defer func() {
			if r := recover(); r != nil {
				log.Print("Error causado por: ", user)
				log.Printf("Recovered from panic: %v", r)
				conn.WriteJSON(ErrorMessage{Action: "error", Message: "Internal server error"})
				proyect.DisconnectUsers()
				rooms.Delete(roomID)
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if permission != 2 {
				var dataMap GeneralMessage
				err := json.Unmarshal([]byte(msg), &dataMap)

				if err != nil {
					log.Print(err)
				}

				if dataMap.Action != "editingUser" && dataMap.Action != "deleteEditingUser" && dataMap.Action != "columns" {
					go func() {
						proyect.mu.Lock()
						saveTimer := proyect.saveTimer
						actionsCounter := proyect.actionsCounter
						proyect.mu.Unlock()

						if saveTimer != nil {
							if actionsCounter >= roomActionsThreshold {
								// Guardado fuera del lock
								err := a.repo.SaveRoom(context.Background(), models.Project{ID: proyect.ID, ProjectInfo: proyect.ProjectInfo, Data: proyect.Data, Config: proyect.Config, Fosil: proyect.Fosil, Facies: proyect.Facies, Muestras: proyect.Muestras, Shared: proyect.Shared})
								if err != nil {
									log.Println("Error guardando el proyecto automáticamente: ", err)

								} else {
									log.Println("Proyecto guardado: ", proyect.ID)
								}

								proyect.mu.Lock()
								proyect.actionsCounter = 0
								proyect.mu.Unlock()

								return
							}

							saveTimer.Reset(5 * time.Minute)

							proyect.mu.Lock()
							proyect.actionsCounter++
							proyect.mu.Unlock()
						} else {
							proyect.mu.Lock()
							proyect.saveTimer = time.NewTimer(5 * time.Minute)
							proyect.actionsCounter = 1
							proyect.mu.Unlock()

							go func() {
								<-proyect.saveTimer.C

								// Guardado fuera del lock
								err := a.repo.SaveRoom(context.Background(), models.Project{ID: proyect.ID, ProjectInfo: proyect.ProjectInfo, Data: proyect.Data, Config: proyect.Config, Fosil: proyect.Fosil, Facies: proyect.Facies, Muestras: proyect.Muestras, Shared: proyect.Shared})
								if err != nil {
									log.Println("Error guardando el proyecto automáticamente: ", err)
								} else {
									log.Println("Proyecto guardado: ", proyect.ID)
								}

								proyect.mu.Lock()
								proyect.saveTimer = nil
								proyect.mu.Unlock()
							}()
						}
					}()
				}

				proyect.mu.Lock()

				switch dataMap.Action {

				case "undo":
					undo(proyect)

				case "redo":
					redo(proyect)

				case "deletetokenLink":

					removeSharedPass(proyect)

				case "generateTokenLink":

					generateTokenLink(conn, roomID, user, proyect)

				case "editingUser":

					var editing dtos.UserEditingState
					err := json.Unmarshal(dataMap.Data, &editing)
					if err != nil {
						log.Println("Error al deserializar: ", err)
					}

					section := editing.Section

					proyect.Active[userID].Editing = section

					msgData := map[string]interface{}{
						"action": "editingUser",
						"value":  section,
						"data": map[string]interface{}{
							"id":    userID,
							"name":  user,
							"color": proyect.Active[userID].Color,
						},
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

					proyect.Active[userID].Editing = ""

					msgData := map[string]interface{}{
						"action":   "deleteEditingUser",
						"value":    section,
						"userName": user,
					}

					sendSocketMessage(msgData, proyect, "deleteEditingUser")

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
				case "drop":
					var drop dtos.Drop
					err := json.Unmarshal(dataMap.Data, &drop)
					if err != nil {
						log.Println("Error al deserializar: ", err)
					}
					activeId := drop.ActiveId
					overId := drop.OverId

					performAction(proyect,
						Action{
							Execute: func() {
								layerDrop(proyect, activeId, overId)
							},
							Undo: func() {
								layerDrop(proyect, overId, activeId)
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
				case "addMuestra":

					var muestra models.Muestra
					err := json.Unmarshal(dataMap.Data, &muestra)
					if err != nil {
						log.Println("Error", err)
						break
					}

					id := shortuuid.New()

					log.Println(muestra, id, "aquiii")

					performAction(proyect,
						Action{
							Execute: func() {
								addMuestra(proyect, id, muestra)
							},
							Undo: func() {
								deleteMuestra(proyect, dtos.DeleteMuestra{IdMuestra: id})
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

					log.Print(editTextData)

					// oldvalue := GetFieldString(proyect.Data[editTextData.RowIndex], editTextData.Key)
					oldvalue, ok := proyect.Data[editTextData.RowIndex].Columns[editTextData.Key].(string)
					if !ok {
						oldvalue = ""
					}
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

				case "editMuestra":

					var muestra dtos.EditMuestra
					err := json.Unmarshal(dataMap.Data, &muestra)
					if err != nil {
						log.Println("Error deserializando muestra:", err)
						break
					}

					oldMuestra := proyect.Muestras[muestra.IdMuestra]

					performAction(proyect,
						Action{
							Execute: func() {
								editMuestra(proyect, muestra.IdMuestra, models.NewMuestra(muestra.Upper, muestra.Lower, muestra.MuestraText, muestra.X))
							},
							Undo: func() {
								editMuestra(proyect, muestra.IdMuestra, oldMuestra)
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

				case "deleteMuestra":

					var muestraID dtos.DeleteMuestra
					err := json.Unmarshal(dataMap.Data, &muestraID)
					if err != nil {
						log.Println("Error deserializando fósil:", err)
						break
					}

					muestra := proyect.Muestras[muestraID.IdMuestra]

					performAction(proyect,
						Action{
							Execute: func() {
								deleteMuestra(proyect, muestraID)
							},
							Undo: func() {
								addMuestra(proyect, muestraID.IdMuestra, muestra)
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
				case "addColumn":

					var column models.Column
					err := json.Unmarshal(dataMap.Data, &column)
					if err != nil {
						log.Println("Error deserializando columna:", err)
						break
					}

					column.Visible = true
					column.Removable = true

					found := false
					for _, col := range proyect.Config.Columns {
						if col.Name == column.Name {
							found = true
							break
						}
					}
					if !found {
						proyect.Config.Columns = append(proyect.Config.Columns, column)

						sendSocketMessage(map[string]interface{}{
							"action": "addColumn",
							"column": column,
						}, proyect, "addColumn")
					}

				case "deleteColumn":

					var column models.Column
					err := json.Unmarshal(dataMap.Data, &column)
					if err != nil {
						log.Println("Error deserializando columna:", err)
						break
					}

					columnDeleted := false

					for i, col := range proyect.Config.Columns {
						if col.Name == column.Name {
							log.Println(col.Removable)
							if col.Removable {
								proyect.Config.Columns = append(proyect.Config.Columns[:i], proyect.Config.Columns[i+1:]...)
								columnDeleted = true
							}
							break
						}
					}

					if columnDeleted {
						sendSocketMessage(map[string]interface{}{
							"action": "deleteColumn",
							"column": column.Name,
						}, proyect, "deleteColumn")
					}

				case "isInverted":

					isInverted(proyect, dataMap)

				case "save":

					a.save(proyect)

				}
				proyect.mu.Unlock()
			} else {
				errMessage := "Error: Don't have permission to edit this document"
				conn.WriteMessage(websocket.TextMessage, []byte(errMessage))
			}
		}
	}

	RemoveElement(a, roomID, userID, proyect)

	return nil
}

// solicitud http para mostrar la cantidad de goroutines
func (a *API) HandleGoroutines(c echo.Context) error {
	return c.String(http.StatusOK, fmt.Sprintf("Número total de goroutines: %d", runtime.NumGoroutine()))
}

func sendSocketMessage(msgData map[string]interface{}, project *RoomData, action string) {
	jsonMsg, err := json.Marshal(msgData)
	if err != nil {
		log.Println("Error serializing message:", err)
		return
	}

	for _, client := range project.Active {
		client.Conn.WriteMessage(websocket.TextMessage, jsonMsg)
	}

}

func (a *API) instanceRoom(ctx context.Context, roomID string) *RoomData {
	// Intenta cargar la sala existente desde sync.Map
	if existingRoom, ok := rooms.Load(roomID); ok {
		return existingRoom.(*RoomData)
	}

	// Si la sala no existe, se crea una nueva instancia de RoomData
	room, err := a.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil
	}

	newRoom := &RoomData{
		ID:          room.ID,
		ProjectInfo: room.ProjectInfo,
		Data:        room.Data,
		Config:      room.Config,
		Fosil:       room.Fosil,
		Facies:      room.Facies,
		Muestras:    room.Muestras,
		Shared:      room.Shared,
		Active:      make(map[string]*UserConnection),
		undoStack:   make([]Action, 0),
		redoStack:   make([]Action, 0),
	}

	// Almacena la nueva sala en el mapa si no existe (control de concurrencia)
	actualRoom, loaded := rooms.LoadOrStore(roomID, newRoom)
	if loaded {
		return actualRoom.(*RoomData) // Si ya existe, usa la instancia creada por otro goroutine
	}

	return newRoom // Devuelve la nueva sala creada
}

func (a *API) HandleGetActiveProject(c echo.Context) error {
	var activeRooms []map[string]interface{}

	// Itera sobre las salas activas de manera segura
	rooms.Range(func(key, value interface{}) bool {
		room := value.(*RoomData)
		room.mu.Lock()
		defer room.mu.Unlock()

		// Extrae los usuarios activos en esta sala
		var users []map[string]string
		for _, userConn := range room.Active {
			users = append(users, map[string]string{
				"email":   userConn.Email,
				"editing": userConn.Editing,
				"color":   userConn.Color,
			})
		}

		// Construye la representación de la sala actual
		roomInfo := map[string]interface{}{
			"roomID": key.(string),
			"users":  users,
		}
		activeRooms = append(activeRooms, roomInfo)
		return true
	})

	return c.JSON(http.StatusOK, activeRooms)
}

// Generar un color aleatorio en formato hexadecimal
func generateRandomColor() string {
	randomIndex := rand.Intn(len(colors))
	return colors[randomIndex]
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

	if rowIndex < -1 || rowIndex > len(project.Data) {
		return
	}

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

	roomData.Columns[key] = value

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

	if newFosil.Upper < 0 || newFosil.Lower < 0 {
		return
	}

	roomData := &project.Fosil
	(*roomData)[id] = newFosil

	msgData := map[string]interface{}{
		"action":  "addFosil",
		"idFosil": id,
		"value":   newFosil,
	}

	sendSocketMessage(msgData, project, "addFosil")

}

func addMuestra(project *RoomData, id string, newMuestra models.Muestra) {

	if newMuestra.Upper < 0 || newMuestra.Lower < 0 {
		return
	}

	log.Println(id, "aquii2")

	roomData := &project.Muestras
	(*roomData)[id] = newMuestra

	log.Println(roomData, "aquii3")

	msgData := map[string]interface{}{
		"action":    "addMuestra",
		"idMuestra": id,
		"value":     newMuestra,
	}

	log.Println(msgData, "aquii4")

	sendSocketMessage(msgData, project, "addMuestra")

}

func deleteFosil(project *RoomData, fosilID dtos.DeleteFosil) {

	id := fosilID.IdFosil
	if _, exists := project.Fosil[id]; !exists {
		return
	}

	roomData := &project.Fosil
	delete(*roomData, id)

	msgData := map[string]interface{}{
		"action":  "deleteFosil",
		"idFosil": id,
	}

	sendSocketMessage(msgData, project, "deleteFosil")

}

func deleteMuestra(project *RoomData, muestraID dtos.DeleteMuestra) {

	id := muestraID.IdMuestra
	if _, exists := project.Muestras[id]; !exists {
		return
	}

	roomData := &project.Muestras
	delete(*roomData, id)

	msgData := map[string]interface{}{
		"action":    "deleteMuestra",
		"idMuestra": id,
	}

	sendSocketMessage(msgData, project, "deleteMuestra")

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

func editMuestra(project *RoomData, id string, newMuestra models.Muestra) {

	roomData := &project.Muestras
	(*roomData)[id] = newMuestra

	msgData := map[string]interface{}{
		"action":    "editMuestra",
		"idMuestra": id,
		"value":     newMuestra,
	}

	sendSocketMessage(msgData, project, "editMuestra")

}

func layerDrop(project *RoomData, activeId int, overId int) {

	roomData := project.Data

	roomData[activeId], roomData[overId] = roomData[overId], roomData[activeId]

	msgData := map[string]interface{}{
		"action":   "drop",
		"activeId": activeId,
		"overId":   overId,
	}

	sendSocketMessage(msgData, project, "drop")

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

	project.UpdateCoord(project.GetSizeRoom())

	msgData := map[string]interface{}{
		"action":     "isInverted",
		"isInverted": isInverted.IsInverted,
		"facies":     project.Facies,
		"fosil":      project.Fosil,
		"muestras":   project.Muestras,
	}

	sendSocketMessage(msgData, project, "isInverted")

}

func (r *RoomData) GetSizeRoom() int {
	var height int
	for _, Lit := range r.Data {
		height += Lit.Litologia.Height
	}

	return height
}

func (room *RoomData) UpdateCoord(totalHeight int) {

	for key, fosil := range room.Fosil {
		fosil.Upper = totalHeight - fosil.Upper
		fosil.Lower = totalHeight - fosil.Lower
		room.Fosil[key] = fosil
	}

	for key, sections := range room.Facies {
		for i := range sections {
			oldY1 := sections[i].Y1
			oldY2 := sections[i].Y2
			sections[i].Y1 = float32(totalHeight) - oldY2
			sections[i].Y2 = float32(totalHeight) - oldY1
		}
		room.Facies[key] = sections
	}

	for key, muestra := range room.Muestras {
		muestra.Upper = int(totalHeight) - muestra.Upper
		muestra.Lower = int(totalHeight) - muestra.Lower
		room.Muestras[key] = muestra
	}
}

func (a *API) save(project *RoomData) {

	err := a.repo.SaveRoom(context.Background(), models.Project{ID: project.ID, ProjectInfo: project.ProjectInfo, Data: project.Data, Config: project.Config, Fosil: project.Fosil, Facies: project.Facies, Muestras: project.Muestras, Shared: project.Shared})
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
	if user == proyect.ProjectInfo.Members.Owner {

		storedpass := proyect.Shared.Pass
		if storedpass == "" {
			var err error
			storedpass, err = generateRandomPass(8)
			if err != nil {
				log.Println("Error generando contraseña aleatoria: ", err)
				return
			}

			proyect.Shared.Pass = storedpass
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

func removeSharedPass(proyect *RoomData) {
	proyect.Shared.Pass = ""
}

func (a *API) ValidateInvitation(c echo.Context) error {
	ctx, claimsAuth, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	var requestBody struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&requestBody); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Verifica el token de invitación
	claims, err := encryption.ParseInviteToken(requestBody.Token)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, "Invalid or expired token link")
	}

	email := claimsAuth["email"].(string)
	storedPass := claims.Pass

	// Acceso seguro a la sala usando sync.Map
	roomInterface, exists := rooms.Load(claims.RoomID)
	var members *models.Members
	var pass string

	if exists {
		existingRoom := roomInterface.(*RoomData)

		// Bloqueo de la sala para operaciones concurrentes internas
		existingRoom.mu.Lock()
		defer existingRoom.mu.Unlock()

		members = &existingRoom.ProjectInfo.Members
		pass = existingRoom.Shared.Pass
	} else {
		// Sala no está en memoria; accede a la base de datos para obtener miembros y pass
		var err error
		members, pass, err = a.repo.GetMembersAndPass(ctx, claims.RoomID)
		if err != nil {
			return a.handleError(c, http.StatusInternalServerError, err.Error())
		}
	}

	// Verifica la validez del token
	if pass != storedPass {
		return a.handleError(c, http.StatusUnauthorized, "Invalid or expired token link")
	}

	// Respuesta de éxito inicial
	response := map[string]interface{}{
		"status": "valid",
		"roomID": claims.RoomID,
		"role":   claims.Role,
	}

	// Comprueba si el usuario ya es miembro
	if members.Owner == email || contains(members.Editors, email) || contains(members.Readers, email) {
		return c.JSON(http.StatusOK, response)
	}

	// Añadir al usuario como miembro si no existe
	if exists {
		existingRoom := roomInterface.(*RoomData)
		switch claims.Role {
		case "editors":
			existingRoom.ProjectInfo.Members.Editors = append(existingRoom.ProjectInfo.Members.Editors, email)
		case "readers":
			existingRoom.ProjectInfo.Members.Readers = append(existingRoom.ProjectInfo.Members.Readers, email)
		}
	}

	// Persistencia en la base de datos
	if err := a.repo.AddUserToProject(ctx, email, claims.Role, claims.RoomID); err != nil {
		return a.handleError(c, http.StatusInternalServerError, "Server error")
	}

	return c.JSON(http.StatusOK, response)
}

func (r *RoomData) AddUser(conn *websocket.Conn, email string, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Active[userID] = &UserConnection{
		Email: email,
		Conn:  conn,
		Color: generateRandomColor(),
	}
}

func (r *RoomData) DisconnectUsers() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Desconectar a todos los usuarios conectados
	for _, client := range r.Active {
		if client != nil {
			message := ErrorMessage{
				Action:  "close",
				Message: "project closed",
			}

			err := client.Conn.WriteJSON(message)
			if err != nil {
				return fmt.Errorf("error sending close message to user %s: %w", client.Email, err)
			}

			client.Conn.Close()
		}
	}

	r.Active = make(map[string]*UserConnection)
	log.Println("All users disconnected from room: ", r.ID)
	return nil
}

func (r *RoomData) DisconnectUser(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, exists := r.Active[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	client.Conn.Close()

	log.Println("\033[35m User disconnected: ", client.Email, "\033[0m")
	delete(r.Active, userID)
	sendSocketMessage(map[string]interface{}{"action": "userDisconnected", "id": userID}, r, "userDisconnected")
	return nil
}

func (r *RoomData) DataProject() map[string]interface{} {

	type User struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	type UserEditing struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Id    string `json:"id"`
	}

	users := make(map[string]User)
	userEditing := make(map[string]UserEditing)

	// Llenar los mapas de usuarios
	for key, value := range r.Active {
		users[key] = User{
			Name:  value.Email,
			Color: value.Color,
		}

		if value.Editing != "" {
			userEditing[value.Editing] = UserEditing{
				Name:  value.Email,
				Color: value.Color,
				Id:    key,
			}
		}
	}

	// Construir el mapa de datos de la sala
	dataRoom := map[string]interface{}{
		"action":      "data",
		"projectInfo": r.ProjectInfo,
		"data":        r.Data,
		"config":      r.Config,
		"fosil":       r.Fosil,
		"muestras":    r.Muestras,
		"facies":      r.Facies,
		"users":       users,
		"userEditing": userEditing,
	}

	return dataRoom
}

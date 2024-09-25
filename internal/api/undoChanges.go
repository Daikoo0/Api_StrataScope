package api

import "log"

type Action struct {
	Execute func() //Apply
	Undo    func() //Revert
}

func performAction(room *RoomData, action Action) {
	action.Execute()
	room.undoStack = append(room.undoStack, action)
	room.redoStack = nil
}

func undo(room *RoomData) {
	if len(room.undoStack) == 0 {
		log.Println("No hay nada que deshacer")
		return
	}

	action := room.undoStack[len(room.undoStack)-1]
	room.undoStack = room.undoStack[:len(room.undoStack)-1]
	action.Undo()
	room.redoStack = append(room.redoStack, action)
}

func redo(room *RoomData) {
	if len(room.redoStack) == 0 {
		log.Println("No hay nada que rehacer")
		return
	}

	action := room.redoStack[len(room.redoStack)-1]
	room.redoStack = room.redoStack[:len(room.redoStack)-1]
	action.Execute()
	room.undoStack = append(room.undoStack, action)
}

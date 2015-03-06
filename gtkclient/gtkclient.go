package main

import (
	//	"github.com/mattn/go-gtk/gdkpixbuf"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	// "os"
	// "os/exec"
	// "path"
	// "regexp"
	// "sort"
	// "strings"
)

type Layout struct {
	// frames + panes
	mainFrame  gtk.IWidget
	leftFrame  gtk.IWidget
	rightFrame gtk.IWidget

	// objects
	editor      *gtk.TextView
	inputEntry  *gtk.Entry
	inputButton *gtk.Button
}

/*
							main
	+----------------------------------------+
	|		left		|		right		|
	|					|					|
	|					|					|
	|					|					|
	|					|					|
	|					+-------------------+
	|					|				|btn|
	+--------------------+-------------------+

*/
func layoutInit() Layout {
	// layout frames
	mainFrame := gtk.NewFrame("")
	mainBox := gtk.NewHBox(true, 1)

	leftFrame := gtk.NewFrame("editor")
	//leftBox := gtk.NewVBox(true, 1)
	leftBox := gtk.NewScrolledWindow(nil, nil)
	leftBox.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	leftBox.SetShadowType(gtk.SHADOW_IN)
	editor := gtk.NewTextView()
	leftBox.Add(editor)

	leftFrame.Add(leftBox)

	// rightFrame := gtk.NewFrame("chat")
	rightPane := gtk.NewVPaned()
	// rightBox := gtk.NewVBox(false, 1)

	chatFrame := gtk.NewFrame("chat")
	//chatBox := gtk.NewVBox(true, 1)
	chatBox := gtk.NewScrolledWindow(nil, nil)
	chatBox.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	chatBox.SetShadowType(gtk.SHADOW_IN)
	chatMessages := gtk.NewTextView()
	chatMessages.SetEditable(false)
	chatMessages.SetCursorVisible(false)
	chatBox.Add(chatMessages)
	chatFrame.Add(chatBox)

	inputFrame := gtk.NewFrame("")
	inputBox := gtk.NewHBox(false, 1)
	inputEntry := gtk.NewEntry()
	inputButton := gtk.NewButtonWithLabel("send")
	inputBox.Add(inputEntry)
	inputBox.Add(inputButton)
	// inputBox.SetSizeRequest(15, 15)
	inputFrame.Add(inputBox)

	//	inputButton.SetSizeRequest(25, 50)

	// rightBox.Add(chatFrame)
	// rightBox.Add(inputFrame)
	//
	// rightFrame.Add(rightBox)

	rightPane.Pack1(chatFrame, false, false)
	rightPane.Pack2(inputFrame, false, false)

	mainBox.Add(leftFrame)
	mainBox.Add(rightPane)
	mainFrame.Add(mainBox)

	inputEntry.SetSizeRequest(450, -1)
	//	inputButton.SetSizeRequest(60, 50)
	chatFrame.SetSizeRequest(500, 550)
	inputFrame.SetSizeRequest(500, 50)

	return Layout{mainFrame, leftFrame, rightPane, editor, inputEntry, inputButton}
}

func main() {
	//var menuitem *gtk.MenuItem
	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("CodeChat")
	window.SetIconName("gtk-dialog-info")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		println("got destroy!", ctx.Data().(string))
		gtk.MainQuit()
	}, "CodeChat")

	layout := layoutInit()

	// try adding an event handler onto the inputButton

	layout.inputButton.Clicked(func() {
		println("button clicked")
	})

	window.Add(layout.mainFrame)
	window.SetSizeRequest(1000, 600)
	window.ShowAll()
	gtk.Main()
}

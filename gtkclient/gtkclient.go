package main

import (
	//	"github.com/mattn/go-gtk/gdkpixbuf"
	"encoding/json"
	//"flag"
	//"fmt"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"log"
	"net"
	//"os"
	//"strings"
)

type Layout struct {
	// frames + panes
	mainFrame  gtk.IWidget
	leftFrame  gtk.IWidget
	rightFrame gtk.IWidget

	// objects
	editor       *gtk.TextView
	inputEntry   *gtk.Entry
	inputButton  *gtk.Button
	chatMessages *gtk.TextView
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
	leftBox := gtk.NewScrolledWindow(nil, nil)
	leftBox.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	leftBox.SetShadowType(gtk.SHADOW_IN)
	editor := gtk.NewTextView()
	leftBox.Add(editor)
	leftFrame.Add(leftBox)
	rightPane := gtk.NewVPaned()
	chatFrame := gtk.NewFrame("chat")
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
	inputFrame.Add(inputBox)
	rightPane.Pack1(chatFrame, false, false)
	rightPane.Pack2(inputFrame, false, false)
	mainBox.Add(leftFrame)
	mainBox.Add(rightPane)
	mainFrame.Add(mainBox)
	inputEntry.SetSizeRequest(450, -1)
	chatFrame.SetSizeRequest(500, 550)
	inputFrame.SetSizeRequest(500, 50)

	return Layout{mainFrame, leftFrame, rightPane, editor, inputEntry, inputButton, chatMessages}
}

type Message struct {
	Cmd string `json:"cmd"`
	Msg string `json:"msg"`
}

type Connect struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
}

func read(conn net.Conn, lyt *Layout) {
	// b := make([]byte, 4096)
	d := json.NewDecoder(conn)
	for {
		var v map[string]interface{}
		err := d.Decode(&v)
		if err != nil {
			log.Println("error, bad json")
			// should exit here : signifies a dead server
			return
		}
		// Catch a response from the server
		if s, ok := v["success"]; ok {
			if s.(bool) {
				log.Println("success")
			} else {
				log.Println("read: command failed")
				log.Println("returned: ", v["status-message"])
			}
			// Catch general messages from the server
		} else if c, ok := v["cmd"]; ok {
			switch c {
			case "message":
				log.Println("got message")
				var end gtk.TextIter
				buffer := lyt.chatMessages.GetBuffer()
				buffer.GetEndIter(&end)
				buffer.Insert(&end, v["payload"].(string)+"\n")
				lyt.inputEntry.SetText("")
			case "client-exit":
				log.Println("client exited")
			case "client-connect":
				log.Println("client entered")
			default:
				log.Println("no cmd parsed. got: ", v)
			}
		} else {
			log.Println("json parsing failed, got: ", v)
		}
	}
}

func main() {
	//var menuitem *gtk.MenuItem

	var name string
	var err error
	var c net.Conn

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
	// try to attach a 'Return' key handler as well

	layout.inputButton.Clicked(func() {
		// send mesage here
		println("send message: ", layout.inputEntry.GetText())
		layout.inputEntry.SetText("")
	})

	layout.inputEntry.Connect("activate", func() {
		msg := layout.inputEntry.GetText()
		println("enter pressed: ", msg)
		var end gtk.TextIter
		buffer := layout.chatMessages.GetBuffer()
		buffer.GetEndIter(&end)
		buffer.Insert(&end, "you: "+msg+"\n")
		layout.inputEntry.SetText("")
		m := Message{"msg", msg}
		//fmt.Println(m)
		b, e := json.Marshal(m)
		if e != nil {
			log.Println("somethin happened...")
		}
		c.Write(b)
	})

	messagedialog := gtk.NewDialog()
	connectBox := messagedialog.GetVBox()
	label := gtk.NewLabel("username")
	username := gtk.NewEntry()
	username.Connect("activate", func() {
		name = username.GetText()
		messagedialog.Destroy()
	})
	connectBox.Add(label)
	connectBox.Add(username)
	messagedialog.AddButton("connect", gtk.RESPONSE_OK)
	messagedialog.Response(func() {
		name = username.GetText()
	})
	label.Show()
	username.Show()
	messagedialog.Run()
	messagedialog.Destroy()
	window.Add(layout.mainFrame)
	window.SetSizeRequest(1000, 600)
	window.ShowAll()

	c, err = net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	go read(c, &layout)

	user := Connect{"connect", name}
	b, err := json.Marshal(user)

	n, err := c.Write(b)
	if err != nil || n == 0 {
		log.Println(err)
		return
	}
	gtk.Main()
}

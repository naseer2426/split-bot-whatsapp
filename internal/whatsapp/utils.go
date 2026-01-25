package whatsapp

import "go.mau.fi/whatsmeow/types/events"

func getMessageText(evt *events.Message) string {
		if extMsg := evt.Message.GetExtendedTextMessage(); extMsg != nil {
			return extMsg.GetText()
		} 
		return evt.Message.GetConversation()
}

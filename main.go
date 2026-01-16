package main

import (
	"context"
	"fmt"
	"log"
	"os"

	// "os/signal"
	// "syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/naseer2426/split-bot-whatsapp/internal/splitbot"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
)

var client *whatsmeow.Client

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so we only log if there's an actual error (not just missing file)
		if _, ok := err.(*os.PathError); !ok {
			log.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}
	
	// Get DATABASE_URL from environment variables
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Println("Error: DATABASE_URL environment variable is not set")
		fmt.Println("Please set it in your .env file or as an environment variable")
		fmt.Println("Example: DATABASE_URL=postgresql://user:password@localhost:5432/whatsapp_bot?sslmode=disable")
		os.Exit(1)
	}
	
	// Set up logging
	dbLog := waLog.Stdout("Database", "INFO", true)
	ctx := context.Background()
	
	// Initialize the PostgreSQL container for storing session data
	container, err := sqlstore.New(ctx, "postgres", databaseURL, dbLog)
	if err != nil {
		panic(err)
	}
	
	// If you want multiple sessions, use container.NewDevice()
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		panic(err)
	}
	
	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	
	// Add event handler for incoming messages
	client.AddEventHandler(eventHandler)
	
	// Connect to WhatsApp
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		
		// Print QR code for scanning
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan this QR code with your WhatsApp app:")
				fmt.Println("1. Open WhatsApp on your phone")
				fmt.Println("2. Tap Menu (⋮) or Settings")
				fmt.Println("3. Tap Linked Devices")
				fmt.Println("4. Tap Link a Device")
				fmt.Println("5. Point your phone at this screen to scan the code")
				fmt.Println("\nQR Code:")
				// Display QR code in terminal using ASCII art
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}
	
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	
	// Listen to Ctrl+C (you can also use other ways to block the main goroutine)
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// <-c
	
	// Query all users from the API
	userList, err := splitbot.GetAllUsers(splitbot.GetAllUsersOptions{
		Limit:  100,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Users: ", len(userList))
	
	// Extract WhatsApp numbers (filtering out nil values)
	pnToUserId := make(map[string]int)
	var whatsappNumbers []string
	for _, user := range userList {
		if user.WhatsappNumber != nil && *user.WhatsappNumber != "" {
			whatsappNumbers = append(whatsappNumbers, *user.WhatsappNumber)
			pnToUserId[*user.WhatsappNumber] = user.ID
		}
	}
	fmt.Println("Whatsapp Numbers: ", len(whatsappNumbers))
	
	resp, err := client.IsOnWhatsApp(ctx, whatsappNumbers)
	if err != nil {
		panic(err)
	}
	
	// Collect all JIDs that are on WhatsApp
	var jidsOnWhatsApp []types.JID
	for _, result := range resp {
		if !result.IsIn {
			fmt.Println(result.JID.User, "is not on WhatsApp")
			continue
		}
		jidsOnWhatsApp = append(jidsOnWhatsApp, result.JID)
	}

	userIdToLid := make(map[int]string)
	
	// Get user info for all JIDs at once
	if len(jidsOnWhatsApp) > 0 {
		userInfo, err := client.GetUserInfo(ctx, jidsOnWhatsApp)
		if err != nil {
			fmt.Printf("Error getting user info: %v\n", err)
		} else {
			// Print detailed information about each result
			for _, result := range resp {
				if !result.IsIn {
					continue
				}
				
				if info, ok := userInfo[result.JID]; ok {
					pn := result.JID.User
					lid := info.LID.User
					userIdToLid[pnToUserId[pn]] = lid
				}
			}
		}
	}

	// Update users' whatsapp_lid using userIdToLid map
	if len(userIdToLid) > 0 {
		// Create a map for quick user lookup
		userMap := make(map[int]splitbot.User)
		for _, user := range userList {
			userMap[user.ID] = user
		}

		// Update each user's whatsapp_lid
		for userId, lid := range userIdToLid {
			if user, ok := userMap[userId]; ok {
				updateReq := splitbot.CreateUserRequest{
					Name:             user.Name,
					Email:            user.Email,
					TelegramUsername: user.TelegramUsername,
					WhatsappNumber:   user.WhatsappNumber,
					WhatsappLID:      &lid,
				}
				
				updatedUser, err := splitbot.UpdateUser(userId, updateReq)
				if err != nil {
					fmt.Printf("Error updating user %d: %v\n", userId, err)
				} else {
					fmt.Printf("Updated user %d (%s) with whatsapp_lid: %s\n", userId, updatedUser.Email, lid)
				}
			}
		}
	}

	client.Disconnect()
}

func eventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		// Only handle text messages
		if evt.Message.GetConversation() != "" || evt.Message.GetExtendedTextMessage() != nil {
			messageText := evt.Message.GetConversation()
			var mentionedJIDs []string
			
			// Check if it's an extended text message (which can contain mentions)
			if messageText == "" && evt.Message.GetExtendedTextMessage() != nil {
				messageText = evt.Message.GetExtendedTextMessage().GetText()
				
				// Extract mentioned JIDs if they exist
				if evt.Message.GetExtendedTextMessage().GetContextInfo() != nil {
					mentionedJIDs = evt.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
				}
			}
			
			fmt.Printf("Received message from %s: %s\n", evt.Info.Sender, messageText)
			if len(mentionedJIDs) > 0 {
				fmt.Printf("  Mentions: %v\n", mentionedJIDs)
			}
			
			// Echo the message back
			echoText := "Echo: " + messageText
			var echoMessage *waProto.Message
			
			// If the message has mentions, use ExtendedTextMessage
			if len(mentionedJIDs) > 0 {
				echoMessage = &waProto.Message{
					ExtendedTextMessage: &waProto.ExtendedTextMessage{
						Text: proto.String(echoText),
						ContextInfo: &waProto.ContextInfo{
							MentionedJID: mentionedJIDs,
						},
					},
				}
			} else {
				// For simple messages without mentions, use Conversation
				echoMessage = &waProto.Message{
					Conversation: proto.String(echoText),
				}
			}
			
			_, err := client.SendMessage(
				context.Background(),
				evt.Info.Chat,
				echoMessage,
			)
			
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
			} else {
				fmt.Printf("Echoed message to %s\n", evt.Info.Chat)
			}
		}
	}
}

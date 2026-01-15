import makeWASocket, {
  DisconnectReason,
  useMultiFileAuthState,
  makeCacheableSignalKeyStore,
  fetchLatestBaileysVersion,
  downloadMediaMessage,
  getContentType,
} from '@whiskeysockets/baileys';
import { Boom } from '@hapi/boom';
import pino from 'pino';
import qrcode from 'qrcode-terminal';
import { createWriteStream, existsSync, mkdirSync } from 'fs';
import { join } from 'path';

// Configure logger
const logger = pino({
  level: 'info',
  transport: {
    target: 'pino-pretty',
    options: {
      colorize: true,
      ignore: 'hostname',
      translateTime: 'SYS:standard',
    },
  },
}).child({ class: 'baileys' });

// Create downloads directory if it doesn't exist
const DOWNLOADS_DIR = './downloads';
if (!existsSync(DOWNLOADS_DIR)) {
  mkdirSync(DOWNLOADS_DIR, { recursive: true });
  console.log(`📁 Created downloads directory: ${DOWNLOADS_DIR}`);
}

async function connectToWhatsApp() {
  // Load auth state from file
  const { state, saveCreds } = await useMultiFileAuthState('auth_info');

  // Fetch latest version info
  const { version, isLatest } = await fetchLatestBaileysVersion();
  console.log(`Using Baileys v${version.join('.')}, isLatest: ${isLatest}`);

  // Initialize socket
  const sock = makeWASocket({
    version,
    logger,
    printQRInTerminal: false, // We'll handle QR display manually
    auth: {
      creds: state.creds,
      keys: makeCacheableSignalKeyStore(state.keys, logger),
    },
    browser: ['WhatsApp Echo Bot', 'Chrome', '10.0'],
    markOnlineOnConnect: true,
  });

  // Handle connection updates
  sock.ev.on('connection.update', async (update) => {
    const { connection, lastDisconnect, qr } = update;

    // Display QR code if available
    if (qr) {
      console.log('\n📱 Scan this QR code with your WhatsApp:\n');
      qrcode.generate(qr, { small: true });
      console.log('\nOpen WhatsApp > Linked Devices > Link a Device\n');
    }

    // Handle connection status
    if (connection === 'close') {
      const shouldReconnect =
        (lastDisconnect?.error as Boom)?.output?.statusCode !==
        DisconnectReason.loggedOut;

      console.log(
        'Connection closed due to',
        lastDisconnect?.error,
        ', reconnecting:',
        shouldReconnect
      );

      if (shouldReconnect) {
        connectToWhatsApp();
      } else {
        console.log('Logged out. Please delete auth_info folder and restart.');
      }
    } else if (connection === 'open') {
      console.log('✅ Connected to WhatsApp!');
      console.log('🤖 Echo bot is now running. Send me a message!');
    }
  });

  // Save credentials whenever they update
  sock.ev.on('creds.update', saveCreds);

  // Handle incoming messages
  sock.ev.on('messages.upsert', async ({ messages, type }) => {
    // Only process new messages
    if (type !== 'notify') return;

    for (const msg of messages) {
      console.log("msg", msg);
      // Ignore if message is from status broadcast or if no message content
      if (!msg.message || msg.key.remoteJid === 'status@broadcast') continue;

      // Get the message type
      const messageType = getContentType(msg.message);
      console.log(`📬 Message type: ${messageType}`);

      // Check if message is from a group and fetch group info
      const remoteJid = msg.key.remoteJid;
      const isGroup = remoteJid?.endsWith('@g.us');
      
      if (isGroup) {
        try {
          const groupMetadata = await sock.groupMetadata(remoteJid!);
          
          console.log('\n👥 GROUP INFORMATION:');
          console.log(`  📛 Name: ${groupMetadata.subject}`);
          console.log(`  🆔 ID: ${groupMetadata.id}`);
          console.log(`  📝 Description: ${groupMetadata.desc || 'No description'}`);
          console.log(`  👤 Owner: ${groupMetadata.owner}`);
          console.log(`  👥 Participants: ${groupMetadata.participants.length}`);
          console.log(`  📅 Created: ${groupMetadata.creation ? new Date(groupMetadata.creation * 1000).toLocaleString() : 'Unknown'}`);
          console.log(`  🔒 Admins only messaging: ${groupMetadata.announce ? 'Yes' : 'No'}`);
          console.log(`  ⚙️  Admins only settings: ${groupMetadata.restrict ? 'Yes' : 'No'}`);
          console.log(`  ➕ Members can add: ${groupMetadata.memberAddMode ? 'Yes' : 'No'}`);
          console.log(`  ✋ Join approval required: ${groupMetadata.joinApprovalMode ? 'Yes' : 'No'}`);
          
          if (groupMetadata.linkedParent) {
            console.log(`  🏘️  Part of community: ${groupMetadata.linkedParent}`);
          }
          
          // Show sender info in group
          const sender = msg.key.participant || msg.key.participantAlt;
          if (sender) {
            const participant = groupMetadata.participants.find(p => 
              p.id === sender || p.id.includes(sender.split('@')[0])
            );
            
            if (participant) {
              console.log(`  📤 Sender: ${sender}`);
              if (participant.admin) {
                console.log(`  ⭐ Sender role: ${participant.admin}`);
              } else {
                console.log(`  ⭐ Sender role: member`);
              }
            }
          }
          
          // List admins
          const admins = groupMetadata.participants.filter(p => p.admin);
          if (admins.length > 0) {
            console.log(`  👑 Admins (${admins.length}):`);
            admins.forEach(admin => {
              console.log(`     - ${admin.id} (${admin.admin})`);
            });
          }
          
          // List all participants
          console.log(`  👥 All Participants (${groupMetadata.participants.length}):`);
          groupMetadata.participants.forEach(participant => {
            const role = participant.admin || 'member';
            const name = participant.notify || participant.name || participant.id;
            const pn = participant.phoneNumber
            console.log(`     - ${name} (${participant.lid}) [${role}] ${pn ? `(${pn})` : ''}`);
          });
          
          console.log(''); // Empty line for readability
          
        } catch (error) {
          console.error('❌ Error fetching group metadata:', error);
        }
      }

      // Handle text messages
      const messageText =
        msg.message.conversation ||
        msg.message.extendedTextMessage?.text ||
        '';

      if (messageText) {
        console.log(`📨 Received: "${messageText}" from ${msg.key.remoteJid}, ${msg.key.participant}, ${msg.key.participantAlt}`);

        try {
          const senderID = msg.key.participant || msg.key.participantAlt || msg.key.remoteJid;
          const senderLidNumber = senderID?.split('@')[0];
          // Send the same message back
          await sock.sendMessage(msg.key.remoteJid!, {
            text: `hey @${senderLidNumber}, you said "${messageText}"`,
            mentions: [senderID!]
          });
          console.log(`✅ Echoed back: "${messageText}"`);
        } catch (error) {
          console.error('❌ Error sending message:', error);
        }
      }

      // Handle image messages
      if (messageType === 'imageMessage') {
        console.log('📷 Image received!');
        
        try {
          // Get image caption if available
          const caption = msg.message.imageMessage?.caption || '';
          console.log(`Caption: ${caption || '(no caption)'}`);
          console.log("type", msg.message.imageMessage?.mimetype);

          // Download image as buffer
          const buffer = await downloadMediaMessage(
            msg,
            'buffer',
            {},
            {
              logger,
              reuploadRequest: sock.updateMediaMessage
            }
          );

          // Convert to base64
          const base64 = buffer.toString('base64');
          console.log(`📦 Image buffer size: ${buffer.length} bytes`);
          console.log(`📝 Base64 length: ${base64.length} characters`);

          // Generate unique filename
          const timestamp = Date.now();
          const filename = `image_${timestamp}_base64.txt`;
          const filepath = join(DOWNLOADS_DIR, filename);

          // Save base64 string to file
          const writeStream = createWriteStream(filepath);
          writeStream.write(base64);
          writeStream.end();

          // Wait for file to be saved
          await new Promise<void>((resolve, reject) => {
            writeStream.on('finish', () => {
              console.log(`✅ Base64 saved: ${filepath}`);
              resolve();
            });
            writeStream.on('error', reject);
          });

          // Echo back confirmation
          await sock.sendMessage(msg.key.remoteJid!, {
            text: `✅ Image received and saved as base64: ${filename}${caption ? `\nOriginal caption: ${caption}` : ''}`
          });

        } catch (error) {
          console.error('❌ Error handling image:', error);
          await sock.sendMessage(msg.key.remoteJid!, {
            text: '❌ Sorry, I had trouble processing that image.'
          });
        }
      }
    }
  });

  return sock;
}

// Start the bot
console.log('🚀 Starting WhatsApp Echo Bot...\n');
connectToWhatsApp().catch((err) => {
  console.error('Failed to connect:', err);
  process.exit(1);
});

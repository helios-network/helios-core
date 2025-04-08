const WebSocket = require('ws')
const { Tendermint34Client } = require('@cosmjs/tendermint-rpc');
const { StargateClient } = require('@cosmjs/stargate');

async function subscribeToEvents() {
  const client = await Tendermint34Client.connect("ws://localhost:26657/websocket");
  
  const subscription = client.subscribeTx(`tm.event='Tx' AND eventType.key='EventTypeDistributeDevRevenue'`);
  
  console.log("Subscribed to revenue distribution events...");
  
  // Handle incoming events
  subscription.addListener({
    next: (event) => {
      console.log("Received revenue distribution event:", event);
      // The event.result.events will contain the Cosmos events
      const attributes = event.result.events
        .filter(e => e.type === "EventTypeDistributeDevRevenue")
        .flatMap(e => e.attributes);
      
      console.log("Attributes:", attributes);
    },
    error: (error) => {
      console.error("Subscription error:", error);
    },
    complete: () => {
      console.log("Subscription completed");
    }
  });
  
  return subscription;
}

// To unsubscribe
// subscription.unsubscribe();


async function monitorEvents() {
	// Create a Tendermint client for WebSocket connections
	const tmClient = await Tendermint34Client.connect("ws://localhost:26657/websocket");
	
	// Create a Stargate client from the Tendermint client
	const client = await StargateClient.create(tmClient);
	
	// Subscribe to new blocks
	const subscription = tmClient.subscribeNewBlock();
	
	subscription.addListener({
	  next: async (blockEvent) => {
		// For each new block, query transactions in that block
		const block = await client.getBlock(blockEvent.header.height);
		
		// For each transaction, check for your revenue events
		for (const txHash of block.txs) {
		  // Convert the binary hash to a hex string
		  const txHashHex = Buffer.from(txHash).toString('hex').toUpperCase();
		  
		  // Query the transaction results
		  const txResult = await client.getTx(txHashHex);
		  
		  // Look for your specific event
		  const revenueEvents = txResult.events.filter(
			event => event.type === "EventTypeDistributeDevRevenue"
		  );
		  
		  if (revenueEvents.length > 0) {
			console.log(`Found revenue events in transaction ${txHashHex}:`, revenueEvents);
		  }
		}
	  },
	  error: (error) => console.error(error),
	  complete: () => console.log("Subscription completed")
	});
	
	return subscription;
}

async function subscribeToRevenueEvents() {
	try {
	  console.log("Connecting to WebSocket...");
	  const client = await Tendermint34Client.connect("ws://localhost:26657/websocket");
	  console.log("Connected successfully!");
	  
	  // Subscribe to all transactions
	  console.log("Subscribing to transactions...");
	  const subscription = client.subscribeTx();
	  
	  subscription.addListener({
		next: (event) => {
		  console.log("Transaction received:", event.hash);
		  
		  if (event.result && event.result.events) {
			const revenueEvents = event.result.events.filter(e => 
			  e.type === "EventTypeDistributeDevRevenue" || 
			  e.type === "revenue_v1.EventTypeDistributeDevRevenue" 
			);
			
			if (revenueEvents.length > 0) {
			  console.log("Found revenue distribution events!", revenueEvents);
			  
			  // Extract attributes
			  const attributes = revenueEvents.flatMap(e => e.attributes);
			  console.log("Event attributes:", attributes);
			}
		  }
		},
		error: (error) => {
		  console.error("Subscription error:", error);
		},
		complete: () => {
		  console.log("Subscription completed");
		}
	  });
	  
	  console.log("Listening for events. Press Ctrl+C to exit.");
	  
	  return subscription;
	} catch (error) {
	  console.error("Failed to set up subscription:", error);
	  throw error;
	}
  }
  

function simpleWebSocketSubscription() {
	
	const ws = new WebSocket("ws://localhost:26657/websocket");
	
	ws.on('open', () => {
		console.log("Direct WebSocket connected!");
	  
		const subscribeMsg = JSON.stringify({
			jsonrpc: "2.0",
			method: "subscribe",
			id: 1,
			params: {
				query: "tm.event='Tx'"
			}
		});
	  
		ws.send(subscribeMsg);
		console.log("Websocket connected and subscribed to transactions events");
	});
	
	ws.on('message', (data) => {
		try {
			const parsed = JSON.parse(data);
		
			// Check if this is an event response
			if (parsed.result && parsed.result.data &&
				parsed.result.data.value &&
				parsed.result.data.value.TxResult) {
		  
				console.log("\n=== New Transaction Detected ===");
				// Extract all events from this transaction
				const events = parsed.result.data.value.TxResult.result.events || [];
		  
				// Check for revenue events with various possible formats
				const revenueEvents = events.filter(e =>
					e.type.includes("Revenue") ||
					e.type.includes("revenue") ||
					e.type === "EventTypeDistributeDevRevenue" ||
					e.type === "revenue_v1.EventTypeDistributeDevRevenue"
				);
		  
				if (revenueEvents.length > 0) {
					console.log("\n🎉 Found revenue distribution events!");
					// console.log(JSON.stringify(revenueEvents, null, 2));
			
					// Extract attributes from events
					revenueEvents.forEach(event => {
						console.log(`\nEvent type: ${event.type}`);
						event.attributes.forEach(attr => {
							const key = decodeAttribute(attr.key);
							const value = decodeAttribute(attr.value);
							console.log(`  ${key}: ${value}`);
						});
					});
				}
			}
		} catch (e) {
			// Just log raw data on parse errors
			console.log("Raw message received:", data.toString().substring(0, 100) + "...");
		}
	});

	return ws;
}  
module.exports = {
	subscribeToEvents,
	monitorEvents,
	subscribeToRevenueEvents,
	simpleWebSocketSubscription
}

function decodeAttribute(data) {
    const approaches = [
        // Approach 1: Standard base64
        () => Buffer.from(data, 'base64').toString(),
        
        // Approach 2: base64url format
        () => Buffer.from(data, 'base64url').toString(),
        
        // Approach 3: UTF-8 encoded strings
        () => Buffer.from(data).toString('utf8'),
        
        // Approach 4: Hex encoded
        () => Buffer.from(data, 'hex').toString()
    ];
    
    // Try each approach until one works
    for (const approach of approaches) {
        try {
            const result = approach();
            // Check if result is reasonably printable
            if (/^[\x20-\x7E]*$/.test(result) && result.length > 0) {
                return result;
            }
        } catch (e) {
            // Continue to next approach
        }
    }
    
    // If all else fails, return the original with a warning
    return `[Undecodable: ${data}]`;
}

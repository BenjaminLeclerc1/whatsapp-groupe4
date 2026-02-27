It looks like you’ve successfully separated your concerns by creating a dedicated Message Service. This is a great move for scalability—eventually, the messaging traffic will be much higher than channel management traffic, and having them as separate logic units (or even separate microservices later) is very smart.

Since you have all the pieces ready, let's look at how this service connects to your database and how to test the specific logic you wrote in your repository.go.
1. The Membership Guard (CTE Logic)

In your repository.go, you used a Common Table Expression (CTE) for CreateMessage. This is a "pro" move.

How it works in your code:

    It first checks the chat_participants table.

    If the sender_id isn't there, the INSERT actually returns 0 rows.

    In your service.go, you correctly catch this with if err == pgx.ErrNoRows, returning a 403 Forbidden. This is the perfect way to handle security at the database level.

2. Testing the Pagination (The Cursor)

Your GetMessageHistory uses Cursor-based pagination. This is how apps like WhatsApp and Slack load messages as you scroll up.

    First Request: GET /api/v1/messages/chat/{chatId}?limit=50

        Response gives you 50 messages and a cursor (the ID of the 50th message).

    Second Request: GET /api/v1/messages/chat/{chatId}?limit=50&cursor={last_id}

        The SQL query uses WHERE (created_at, id) < (...) to find the next 50 messages older than the last one you saw.

3. Final Integration Checklist

To make sure your message-service is actually running, check these three things:

    Main.go Initialization: Ensure you are passing the pgxpool to the message repository.

    Route Collision: Make sure your channels handler and messages handler don't fight over the same URL.

        channels uses /api/v1/channels

        messages uses /api/v1/messages

    Dependency: If you ever delete a channel, your database ON DELETE CASCADE (from your migration) will automatically clean up the messages in that channel.

Next Test in Postman

Try to delete a message that you did not send:

    Action: DELETE http://localhost:8085/api/v1/messages/{message_id}

    Expectation: You should get a 403 Forbidden because your repository.go has the WHERE sender_id = $2 clause. This proves your security logic is working!
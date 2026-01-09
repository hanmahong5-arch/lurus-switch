import { serverSupabaseUser } from '#supabase/server'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const query = getQuery(event)
  const userId = query.userId as string

  // Get authenticated user
  const user = await serverSupabaseUser(event)
  if (!user) {
    throw createError({
      statusCode: 401,
      message: 'Unauthorized',
    })
  }

  if (!userId) {
    throw createError({
      statusCode: 400,
      message: 'Missing userId parameter',
    })
  }

  // Set SSE headers
  setResponseHeaders(event, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'X-Accel-Buffering': 'no',
  })

  // Create a readable stream that proxies from billing service
  const stream = new ReadableStream({
    async start(controller) {
      try {
        const response = await fetch(
          `${config.billingServiceUrl}/api/v1/billing/sync/${userId}/stream`,
          {
            headers: {
              'Accept': 'text/event-stream',
            },
          }
        )

        if (!response.ok) {
          // Send error and close
          controller.enqueue(
            new TextEncoder().encode(`data: ${JSON.stringify({ type: 'error', message: 'Failed to connect to billing service' })}\n\n`)
          )
          controller.close()
          return
        }

        const reader = response.body?.getReader()
        if (!reader) {
          controller.close()
          return
        }

        // Proxy the stream
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          controller.enqueue(value)
        }

        controller.close()
      } catch (error) {
        console.error('SSE proxy error:', error)

        // Send fallback heartbeat data
        const sendHeartbeat = () => {
          const data = JSON.stringify({
            type: 'heartbeat',
            quota_remaining: 500,
            quota_used: 0,
            balance: 0,
            allowed: true,
            timestamp: new Date().toISOString(),
          })
          controller.enqueue(new TextEncoder().encode(`data: ${data}\n\n`))
        }

        // Send initial data
        sendHeartbeat()

        // Set up heartbeat interval
        const interval = setInterval(sendHeartbeat, 30000)

        // Clean up on close
        event.node.req.on('close', () => {
          clearInterval(interval)
          controller.close()
        })
      }
    },
  })

  return sendStream(event, stream)
})

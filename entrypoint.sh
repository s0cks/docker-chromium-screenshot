#!/usr/bin/env sh
npm run serve &
SERVER_PID=$!

echo "Waiting for Express server to start..."
while ! nc -z localhost 3000; do
  sleep 0.5
done
echo "Express server is ready!"

npm run build

npm run capture
EXIT_CODE=$?

echo "Stopping Express server..."
kill $SERVER_PID

chmod 666 /app/out
exit $EXIT_CODE

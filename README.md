# docker-chromium-screenshot

Create a screenshot of a SPA using Playwright and Chromium in Docker.

Useful for creating dense info-graphics using web technologies.

## Building

```sh
docker buildx . \
    -t ghcr.io/s0cks/docker-chromium-screenshot:latest
```

## Running

```sh
docker run \
    -v $(pwd)/out:/out \
    ghcr.io/s0cks/docker-chromium-screenshot:latest
```

### Process

```mermaid
sequenceDiagram
    autonumber
    participant Shell as Container Shell
    participant Exp as Express Server
    participant Cap as Capture Task
    participant FS as Container FS

    Shell->>Exp: Start in background (npm run serve &)

    loop Until Port 3000 is Open
        Shell->>Exp: nc -z localhost 3000
        Note over Shell: Sleep 0.5s if closed
    end

    Shell->>Cap: Run capture (npm run capture)
    Cap->>FS: Save screenshot.png
    Cap-->>Shell: Return EXIT_CODE

    Shell->>Exp: Stop server (kill $SERVER_PID)
    Note over Shell: Exit with EXIT_CODE
```

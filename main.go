package main

import (
    "log"
    "github.com/mikan-panel/sdk"
    "github.com/Fortumers/Addon-Mikan/addon"
)

func main() {
    // Register the addon with the Mikan panel SDK
    if err := addon.Register(); err != nil {
        log.Fatalf("failed to register addon: %v", err)
    }

    // Start the SDK runtime – it will keep the process alive and handle events.
    if err := sdk.Start(); err != nil {
        log.Fatalf("sdk start error: %v", err)
    }
}

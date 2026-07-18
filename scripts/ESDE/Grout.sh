#!/bin/bash
# Grout launcher for ES-DE setups (vanilla ES-DE, EmuDeck, RetroDECK).
# Install into your ES-DE ROMs "ports" folder, e.g.:
#   vanilla ES-DE: ~/ROMs/ports/
#   EmuDeck:       ~/Emulation/roms/ports/
#   RetroDECK:     ~/retrodeck/roms/ports/
CUR_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$CUR_DIR/Grout"
LOCK_DIR="${XDG_RUNTIME_DIR:-/tmp}/grout-esde.lock"

# ES-DE can double-fire launch events; make sure only one Grout runs.
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    if [ -f "$LOCK_DIR/pid" ] && kill -0 "$(cat "$LOCK_DIR/pid")" 2>/dev/null; then
        exit 0
    fi
    rm -rf "$LOCK_DIR"
    mkdir "$LOCK_DIR" 2>/dev/null || exit 0
fi
printf '%s\n' "$$" > "$LOCK_DIR/pid"
trap 'rm -rf "$LOCK_DIR"' EXIT

# Apply pending update
if [ -d "$CUR_DIR/.update" ]; then
    cp -rf "$CUR_DIR/.update/"* "$CUR_DIR/"
    rm -rf "$CUR_DIR/.update"
fi

cd "$APP_DIR" || exit 1

export CFW=ESDE
# Grout lives in <roms>/ports/, so the parent of this script's directory is the
# ES-DE ROMs directory regardless of variant or custom install location.
export GROUT_ESDE_ROMS_DIR="$(cd "$CUR_DIR/.." && pwd)"
export LD_LIBRARY_PATH="$APP_DIR/lib:$LD_LIBRARY_PATH"

# Face-button orientation is handled by Grout's bundled ES-DE input mapping
# (direct A=A/B=B for the Deck's Xbox layout) and the in-app "Swap Face Buttons"
# setting. Do NOT export FLIP_FACE_BUTTONS here: the env var overrides and pins
# that setting, making the in-app toggle a no-op.

# Prefer SDL's game controller API and ignore duplicate keyboard/raw joystick events.
export DISABLE_KEYBOARD_INPUT=1
export DISABLE_JOYSTICK_INPUT=1

# Teach SDL about controllers newer than its built-in database (e.g. the current
# Steam Controller) so they're recognized as game controllers. Under Steam Input
# in Game Mode the device is already a virtual Xbox pad and this is a harmless
# no-op; it matters when running the raw controller in Desktop Mode.
if [ -f "$APP_DIR/gamecontrollerdb.txt" ]; then
    export SDL_GAMECONTROLLERCONFIG_FILE="$APP_DIR/gamecontrollerdb.txt"
fi

# Grout uses its own gamepad-driven keyboard. On Steam Deck, SDL/Steam can
# also surface the Steam keyboard, which can interfere with Grout's keyboard.
export SDL_ENABLE_SCREEN_KEYBOARD=0
export SDL_IME_SHOW_UI=0

chmod +x ./grout
./grout

# ES-DE has no restart hook; consume any stale restart flag other CFWs use.
rm -f "$APP_DIR/es_restart_request"

exit 0

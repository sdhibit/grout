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

# Steam Deck uses Xbox-style face buttons, so use direct A=A/B=B mappings.
export FLIP_FACE_BUTTONS=1

# Prefer SDL's game controller API and ignore duplicate keyboard/raw joystick events.
export DISABLE_KEYBOARD_INPUT=1
export DISABLE_JOYSTICK_INPUT=1

# Grout uses its own gamepad-driven keyboard. On Steam Deck, SDL/Steam can
# also surface the Steam keyboard, which can interfere with Grout's keyboard.
export SDL_ENABLE_SCREEN_KEYBOARD=0
export SDL_IME_SHOW_UI=0

chmod +x ./grout
./grout

# ES-DE has no restart hook; consume any stale restart flag other CFWs use.
rm -f "$APP_DIR/es_restart_request"

exit 0

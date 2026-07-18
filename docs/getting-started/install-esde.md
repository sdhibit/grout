# Installation Guide for ES-DE (EmuDeck / RetroDECK / vanilla)

This guide will help you install Grout on desktop Linux setups built around
[ES-DE][esde] (EmulationStation Desktop Edition), including [EmuDeck][emudeck]
and [RetroDECK][retrodeck] on the Steam Deck, Steam Machine, SteamOS, and
Bazzite.

Grout installs as an ES-DE **port**: it lives in your ROMs `ports` folder and
is launched from ES-DE's Ports system.

## Tested Devices

| Manufacturer | Device                        |
|--------------|-------------------------------|
| _None yet_   | _Please report your results!_ |

_Please help verify compatibility on other devices by reporting your results!_

## Installation Steps

1. Download the [latest Grout release](https://github.com/rommapp/grout/releases/latest/download/Grout-ESDE.zip) for ES-DE.
2. Unzip the downloaded archive.
3. Copy the `Grout` folder and the `Grout.sh` file into your ES-DE `ports` ROM
   folder:
    - **Vanilla ES-DE**: `~/ROMs/ports/` (or your custom ROMs directory)
    - **EmuDeck**: `~/Emulation/roms/ports/` (or on your SD card, e.g.
      `/run/media/<card>/Emulation/roms/ports/`)
    - **RetroDECK**: `~/retrodeck/roms/ports/`
4. Make sure `Grout.sh` is executable. Extracting the zip with `unzip` or a
   Linux archive tool preserves this, but extracting on Windows or transferring
   via SMB/MTP loses it — if Grout doesn't launch from ES-DE, run:
   ```sh
   chmod +x Grout.sh
   ```
5. Restart ES-DE (or use *Utilities → Rescan ROM directory*) so the Ports
   system picks up Grout.
6. Launch Grout from the `Ports` menu.
7. On first launch, Grout asks which ES-DE setup you are using (**ES-DE**,
   **EmuDeck**, or **RetroDECK**). Grout pre-selects the variant it detects on
   your system; confirm or change it. This sets sensible defaults for your ROM,
   BIOS, save, and media directories.

!!! tip "Custom install locations"
    EmuDeck and RetroDECK both allow moving their install root (e.g. to an SD
    card). Grout reads EmuDeck's `settings.sh`, RetroDECK's `retrodeck.json`,
    and ES-DE's `es_settings.xml` to find your actual directories. If anything
    still points to the wrong place, every directory can be overridden in
    *Settings → ES-DE Settings*.

## Directory Defaults

| Directory   | Vanilla ES-DE                | EmuDeck                             | RetroDECK                  |
|-------------|------------------------------|-------------------------------------|----------------------------|
| ROMs        | `~/ROMs`                     | `~/Emulation/roms`                  | `~/retrodeck/roms`         |
| BIOS        | `~/.config/retroarch/system` | `~/Emulation/bios`                  | `~/retrodeck/bios`         |
| Saves       | `~/.config/retroarch/saves`  | `~/Emulation/saves`                 | `~/retrodeck/saves`        |
| ES-DE data  | `~/ES-DE`                    | `~/ES-DE`                           | RetroDECK flatpak config   |
| Media       | `~/ES-DE/downloaded_media`   | `~/Emulation/tools/downloaded_media`| RetroDECK media directory  |

Downloaded games appear in ES-DE after a restart or a ROM directory rescan —
ES-DE has no external refresh hook Grout can trigger.

Game metadata is written to ES-DE gamelists (`<ES-DE data>/gamelists/`), and
artwork goes into ES-DE's `downloaded_media` layout (covers, marquees,
backcovers, fan art, videos, manuals), so everything shows up natively in
ES-DE's UI.

## Update

### In-App update (Recommended)

Grout has a built-in update mechanism. To update Grout, launch the application and navigate to the `Settings` menu. From there,
select `Check for Updates`. If a new version is available, follow the on-screen prompts to download and install the update.

### Manual update

To update Grout, simply download the latest release and replace the existing Grout folder in your `ports` directory. If you
have made any custom configurations, ensure to back them up before replacing the folder. Be sure to keep the `config.json`
file if you do not want to authenticate again, and configure platforms folder mappings again.

## Next Steps

After installation is complete, check out the [User Guide](../usage/guide.md) to learn how to use Grout.

--8<-- "docs/_includes/cfw-links.md"

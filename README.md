# Virtual Flight Watcher

A Discord bot that lets you look up live flight and ATC information from the **VATSIM** and **IVAO** networks directly in your Discord server, with a route map for every flight.

---

## Features

- **Both networks at once** — every lookup searches VATSIM and IVAO and shows a result for each network where the pilot or controller is found (or restrict it to one network with the `network` option)
- **Pilot lookup** — real-time position, altitude, speed, QNH, flight plan and more
- **Route map** — a map of the planned route and the aircraft's current position, including SID / STAR and airway expansion when navdata is installed
- **Controller lookup** — active ATC stations with frequency and ATIS
- **ATIS lookup** — the current ATIS of any active station
- **Prefile lookup** — filed flight plans before departure (VATSIM only)
- **Facility / Rating info** — VATSIM facilities, ATC ratings, pilot ratings and military ratings
- **Per-server configuration** — language, units, private replies and channel restriction, set with a single `/config` command
- **Multilingual** — English, Français, Español, Deutsch, Português (Brasil), Italiano, Русский and Türkçe
- **Live data** — network data is refreshed every 15 seconds in the background with zero impact on response time

---

## Commands

| Command | Description | Options |
|---|---|---|
| `/pilot` | Get live pilot info | `cid_or_callsign`, `network` (optional) |
| `/controller` | Get live controller info | `cid_or_callsign`, `network` (optional) |
| `/atis` | Get the ATIS of a station | `cid_or_callsign`, `network` (optional) |
| `/prefile` | Get a prefiled flight plan (VATSIM only) | `cid_or_callsign`, `network` (optional) |
| `/facility` | Get a VATSIM facility | `id_or_name` (e.g. `5` or `APP`) |
| `/rating` | Get a VATSIM ATC rating | `id_or_name` (e.g. `3` or `S2`) |
| `/pilotrating` | Get a VATSIM pilot rating | `id_or_name` |
| `/militaryrating` | Get a VATSIM military rating | `id_or_name` |
| `/config` | Show or change the bot settings of the server | see below |

> **Tips**
> - Callsigns are not case sensitive: `afr123` and `AFR123` give the same result.
> - Without the `network` option, both VATSIM and IVAO are searched.
> - For `/atis`, you don't need to include `_ATIS` — just type the station name (e.g. `LFPG` or `LFPG_APP`) and the bot handles it automatically.

### `/config`

Requires the **Manage Server** permission. Every option is optional: `/config` alone shows the current configuration, and several options can be changed in a single call (e.g. `/config language:Français units:Metric`). In DMs, the settings only apply to you.

| Option | Description |
|---|---|
| `language` | Bot language |
| `ephemeral` | Replies are only visible to the person who ran the command |
| `units` | `Aviation` (ft, kts, inHg + hPa, nm), `Metric` (m, km/h, hPa, km) or `Imperial` (ft, mph, inHg, mi) |
| `channel` | Restrict bot commands to one text channel |
| `all_channels` | Remove the channel restriction |
| `reset` | Reset every setting to its default value |

Settings are saved per server in `data/settings.json` (change the location with `SETTINGS_PATH`).

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/) 1.25+
- A Discord bot token ([create one here](https://discord.com/developers/applications)). The invite link needs the `bot` and `applications.commands` scopes.
- A [Mapbox](https://www.mapbox.com/) access token (optional, for the route maps)
- The Navigraph navdata (optional, for SID / STAR and airway expansion on the maps)

### Installation

```bash
git clone https://github.com/Skyfield1888/virtualflightwatcher.git
cd virtualflightwatcher
```

### Configuration

Create the `.env` file at the root of the project:

```env
DISCORD_TOKEN=your_discord_bot_token_here
MAPBOX_TOKEN=your_mapbox_token_here
```

| Variable | Required | Description |
|---|---|---|
| `DISCORD_TOKEN` | yes | Discord bot token |
| `MAPBOX_TOKEN` | no | Without it, no map is shown |
| `SETTINGS_PATH` | no | Where the per-server settings are saved (default `data/settings.json`) |

> Never commit your `.env` file.

### Navdata (optional)

The route maps can draw airways, SIDs and STARs from Navigraph data. In the `navdata` folder, you will need the latest content of the `navdata_native` zip in the last Navigraph AIRAC, unzipped directly into the folder.

The bot reads `airports.dat`, `wpnavfix.txt`, `wpnavaid.txt`, `wpnavrte.txt` and the `<icao>.xml` procedure file of each airport from it. Without navdata, the route is drawn as straight lines between the airports and the lat/lon waypoints of the route.

> The Navigraph data is proprietary: the `navdata/` folder is git-ignored and must never be committed. Update it at every new AIRAC cycle and restart the bot.

### Running

```bash
go run main.go
```

Slash commands are registered globally on startup. Discord can take a few minutes to show new or changed commands.

---

## Project Structure

```
virtualflightwatcher/
├── main.go
├── api/
│   ├── geo/
│   │   ├── geo.go            # airports, geometry, Mapbox static map URL
│   │   ├── navdata.go        # Navigraph navdata, route and airway expansion
│   │   ├── procs.go          # SID / STAR procedures
│   │   └── weather.go        # QNH from real-world METARs
│   ├── ivao/
│   │   └── ivao_api.go
│   └── vatsim/
│       └── vatsim_api.go
├── config/
│   └── config.go             # per-server settings, saved as JSON
├── i18n/
│   ├── i18n.go               # languages and translation lookup
│   └── lang_*.go             # one file per language
├── discord_utils/
│   ├── commands/
│   │   ├── commands.go       # slash commands and embeds
│   │   ├── settings.go       # settings, units, dispatch, multi-network lookups
│   │   └── config_cmd.go     # the /config command
│   └── events/
│       └── events.go
├── navdata/                  # Navigraph data (not committed)
├── data/                     # generated settings (not committed)
├── go.sum
└── go.mod
```

---

## How It Works

On startup, the bot fetches the full VATSIM data feed (`https://data.vatsim.net/v3/vatsim-data.json`) and the IVAO public data feed (`https://api.ivao.aero/v2/tracker/whazzup`) and indexes every pilot, controller, ATIS, prefile*, facility* and rating into in-memory hash maps. A background goroutine refreshes the data every 15 seconds. All Discord command lookups are instant O(1) map reads — no API call is made per command.

"*" : VATSIM only

IVAO's feed carries no pressure, so the QNH shown for IVAO pilots comes from the real-world METAR (aviationweather.gov) of the departure or arrival airport nearest to the aircraft.

Route maps are generated as [Mapbox](https://www.mapbox.com/) static images: the route is resolved from the flight plan (fixes, navaids, airways, lat/lon waypoints, SID and STAR), drawn as great-circle legs, and the aircraft's current position is added when known.

---

## Dependencies

| Package | Purpose |
|---|---|
| [`bwmarrin/discordgo`](https://github.com/bwmarrin/discordgo) | Discord API client |
| [`joho/godotenv`](https://github.com/joho/godotenv) | `.env` file loading |

External services: VATSIM and IVAO data feeds, [Mapbox](https://www.mapbox.com/) static images (© Mapbox © OpenStreetMap), aviationweather.gov METARs, Navigraph navdata.

---

## Roadmap

- [x] IVAO network support
- [x] Flight tracking / route map
- [x] Translations and bot settings
- [ ] Event notifications (VATSIM and IVAO events)
- [ ] Prefile notifications on new prefiles for a set facility, for controllers

---

## License

Released under the GNU General Public License v3.0, see [LICENSE](LICENSE).
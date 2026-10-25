# Virtual Flight Watcher

A Discord bot that lets you look up live flight and ATC information from the **VATSIM** and **IVAO** networks directly in your Discord server, with a route map for every flight, and that can announce the upcoming network events as native Discord events.

---

## Features

- **Both networks at once** — every lookup searches VATSIM and IVAO and shows a result for each network where the pilot or controller is found (or restrict it to one network with the `network` option)
- **Pilot lookup** — real-time position, altitude, speed, QNH, flight plan and more
- **Route map** — a map of the planned route and the aircraft's current position, including SID / STAR and airway expansion when navdata is installed
- **Controller lookup** — active ATC stations with frequency and ATIS
- **ATIS lookup** — the current ATIS of any active station
- **Prefile lookup** — filed flight plans before departure (VATSIM only)
- **Facility / Rating info** — VATSIM facilities, ATC ratings, pilot ratings and military ratings
- **Event announcements** — upcoming VATSIM and/or IVAO events are created automatically as Discord scheduled events (opt-in, per server, you choose the network)
- **Per-server configuration** — language, units, private replies, channel restriction and event announcements, set with a single `/config` command
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
| `events` | Which network's events are announced as Discord events: `VATSIM`, `IVAO`, `VATSIM + IVAO` or `None` (servers only) |
| `reset` | Reset every setting to its default value (this also disables event announcements) |

Settings are saved per server in `data/settings.json` (change the location with `SETTINGS_PATH`).

### Event announcements

Event announcements are **disabled by default** (`None`). A server administrator chooses the network(s) with:

```
/config events:VATSIM
/config events:IVAO
/config events:VATSIM + IVAO
/config events:None
```

- The bot fetches the events of the selected network(s) every 15 minutes and creates one Discord scheduled event (external type) per upcoming event, prefixed with `[VATSIM]` or `[IVAO]`.
- Changing the choice applies immediately: switching from `VATSIM + IVAO` to `VATSIM` deletes the bot's IVAO events, and switching to a new network creates its events right away.
- Only events starting within the next 14 days are announced (Discord allows 100 scheduled events per server).
- Each event shows the airports, the organisers and a link to the event page, in the language of the server. HTML in the original descriptions is converted to plain text.
- Duplicates are avoided without any database: every created event ends with a `ref:<network>-<id>` marker that the bot looks for before creating a new one.
- `/config events:None` (or `/config reset:True`) deletes every event created by the bot on the server. Events created by humans are never touched.
- The bot needs the **Manage Events** permission on the server.

> If a moderator manually deletes an event created by the bot while it is still in the network feed, it will be recreated at the next sync.
>
> IVAO events are fetched on a best-effort basis: depending on the IVAO API, an `IVAO_API_KEY` may be required (see below). If nothing shows up for IVAO, check the bot logs.

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/) 1.25+
- A Discord bot token ([create one here](https://discord.com/developers/applications)). The invite link needs the `bot` and `applications.commands` scopes, and the **Manage Events** permission if you want event announcements.
- A [Mapbox](https://www.mapbox.com/) access token (optional, for the route maps)
- The Navigraph navdata (optional, for SID / STAR and airway expansion on the maps)

### Installation

```bash
git clone https://github.com/Skyfield1888/virtualflightwatcher.git
cd virtualflightwatcher
```

### Configuration

Copy the provided `.env.example` to `.env` at the root of the project and fill it in:

```bash
cp .env.example .env
```

```env
DISCORD_TOKEN=your_discord_bot_token_here
MAPBOX_TOKEN=your_mapbox_token_here
IVAO_API_KEY=
SETTINGS_PATH=
```

| Variable | Required | Description |
|---|---|---|
| `DISCORD_TOKEN` | yes | Discord bot token |
| `MAPBOX_TOKEN` | no | Without it, no map is shown |
| `IVAO_API_KEY` | no | API key sent to the IVAO events endpoint, if it requires one. Without it, IVAO events may not be available |
| `SETTINGS_PATH` | no | Where the per-server settings are saved (default `data/settings.json`) |

> Never commit your `.env` file (it is git-ignored). Only `.env.example`, which contains no secret, is committed.

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
├── .env.example              # template of the .env file
├── api/
│   ├── geo/
│   │   ├── geo.go            # airports, geometry, Mapbox static map URL
│   │   ├── navdata.go        # Navigraph navdata, route and airway expansion
│   │   ├── procs.go          # SID / STAR procedures
│   │   └── weather.go        # QNH from real-world METARs
│   ├── ivao/
│   │   ├── ivao_api.go
│   │   └── ivao_events.go    # IVAO events feed
│   └── vatsim/
│       ├── vatsim_api.go
│       └── vatsim_events.go  # VATSIM events feed
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
│       └── events.go         # sync of VATSIM / IVAO events to Discord scheduled events
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

Events are handled by a separate background loop: every 15 minutes it fetches the VATSIM events (`https://my.vatsim.net/api/v1/events/all`) and the IVAO events, then, for each server that enabled them, creates the missing Discord scheduled events. The feeds are cached for 2 minutes so several servers never trigger several requests.

---

## Dependencies

| Package | Purpose |
|---|---|
| [`bwmarrin/discordgo`](https://github.com/bwmarrin/discordgo) | Discord API client |
| [`joho/godotenv`](https://github.com/joho/godotenv) | `.env` file loading |

External services: VATSIM and IVAO data and events feeds, [Mapbox](https://www.mapbox.com/) static images (© Mapbox © OpenStreetMap), aviationweather.gov METARs, Navigraph navdata.

---

## Roadmap

- [x] IVAO network support
- [x] Flight tracking / route map
- [x] Translations and bot settings
- [x] Event announcements (VATSIM and IVAO events)
- [x] Choose the network(s) of the announced events (VATSIM / IVAO / both)

---

## Legal

- [Terms of Service](TERMS.md)
- [Privacy Policy](PRIVACY.md)

### License

Virtual Flight Watcher is free software: you can redistribute it and/or modify it under the terms of the **GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version** (SPDX: `GPL-3.0-or-later`). The full text of version 3 is in [LICENSE](LICENSE).

This program is distributed in the hope that it will be useful, but **WITHOUT ANY WARRANTY**; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details. You should have received a copy of the GNU General Public License along with this program. If not, see <https://www.gnu.org/licenses/>.

In practice, this means you are free to use, study, modify and share the code, as long as any version you distribute is also released under the GPL (version 3 or, at your option, any later version) and its source code is made available. This license covers the code only; the use of the hosted bot is covered by the [Terms of Service](TERMS.md).

Third-party data is **not** covered by this license: the Navigraph navdata is proprietary and is not distributed with the project, and the VATSIM / IVAO data, Mapbox maps and aviationweather.gov METARs remain subject to their own terms.
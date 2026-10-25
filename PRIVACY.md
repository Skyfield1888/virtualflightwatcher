# Privacy Policy — Virtual Flight Watcher

*Last updated: October 25, 2026*

This Privacy Policy explains what data the Discord bot **Virtual Flight Watcher** ("the Bot") processes, why, and what your rights are. By using the Bot you agree to this policy.

## 1. Who is responsible

The Bot is operated by **Skyfield1888** ("the operator").
Contact: **https://github.com/Skyfield1888/virtualflightwatcher/issues**

## 2. Data we process

### a) Bot settings (stored)
When a server administrator uses `/config`, the Bot stores in a file on its host:
- the Discord **server ID** (or, when `/config` is used in DMs, `user:` followed by your Discord **user ID**);
- the chosen language, units, "private replies" option, the ID of the allowed channel (if any), and the events mode (`none`, `vatsim`, `ivao`, `both`).

Purpose: to apply your preferences. These data are kept until you reset them (`/config reset:True`) or ask for their deletion.

### b) Command input (not stored)
When you use a command (e.g. `/pilot`), the Bot receives the text you typed (a callsign, a CID, an ICAO code…) together with technical Discord identifiers (server, channel and user IDs). They are used only to answer your command and are **not stored**. The Bot does not read the content of your other messages.

### c) Public network data (not stored on disk)
The Bot downloads the public data feeds of **VATSIM** and **IVAO** every 15 seconds and keeps them in memory only. These feeds are published by the networks and contain data such as pilot or controller names, network IDs (CID / user ID), callsigns, positions and flight plans. The Bot only displays them in answers to commands. We are not the source of this data: to correct or remove it, contact the network concerned.

### d) Events
If an administrator enables event announcements, the Bot creates Discord scheduled events from public VATSIM / IVAO event information (name, description, airports, organisers, dates, link). No personal data of server members is involved.

### e) Technical logs
The Bot may write technical logs (errors, failed lookups, route-resolution messages, events synchronisation). They may contain public flight-plan information or technical Discord identifiers. They are kept only on the host, for troubleshooting, and for no longer than **[e.g. 30 days / until log rotation]**.

We do **not** sell your data, do not use it for advertising or profiling, and do not track you across services.

## 3. Third parties

To provide the service, some data is exchanged with:
- **Discord** — the platform on which the Bot runs ([privacy policy](https://discord.com/privacy)).
- **VATSIM** and **IVAO** — the Bot requests their public data and event feeds.
- **Mapbox** — route maps are static images whose URL contains the route and, for a live flight, the aircraft position (all public flight data). The image is fetched by Discord's servers/clients from Mapbox ([privacy policy](https://www.mapbox.com/legal/privacy)).
- **aviationweather.gov** (NOAA) — the Bot requests the METAR of an airport (only an ICAO code is sent) to display the QNH on IVAO flights.

The Bot does not send your Discord IDs to VATSIM, IVAO, Mapbox or aviationweather.gov.

## 4. Retention and deletion

- Settings: kept until reset with `/config reset:True`, or deleted on request.
- Network data: in memory only, replaced every 15 seconds.
- Logs: see section 2 e).

To request deletion of the data linked to your server or account, contact the operator (section 1).

## 5. Your rights

If you are in the European Union / EEA or the UK, you have the right to access, rectify, erase, restrict or object to the processing of your personal data, and the right to lodge a complaint with a supervisory authority (in France: the CNIL, [cnil.fr](https://www.cnil.fr)). The legal basis of the processing is our legitimate interest in running the Bot you chose to use. To exercise your rights, contact the operator.

## 6. Security

Settings are stored on a private host, the Discord token and API keys are kept out of the public source code, and the Bot only requests the permissions it needs. No system is perfectly secure, however, and we cannot guarantee absolute security.

## 7. Children

The Bot is intended for Discord users, who must meet Discord's minimum age requirement. We do not knowingly collect data from children under 13.

## 8. Changes

This policy may be updated. The date at the top shows the latest version; the history is available in the project's repository.

## 9. Contact

**https://github.com/Skyfield1888/virtualflightwatcher/issues**
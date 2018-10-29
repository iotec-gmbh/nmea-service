# nmea-service

Microservice to provide GPS Data

## Installation

    $ go get -v github.com/iotec-gmbh/nmea-service

## Start

    $ ./nmea-service --help
    usage: nmea-service [<flags>]

    Flags:
      --help                Show context-sensitive help (also try --help-long and --help-man).
      --verbose             Enable verbose mode.
      --tty="/dev/ttyUSB0"  Serial Connection.
      --baudrate=115200     Baudrate of the Serial Connection.
      --host="localhost"    Host to listen.
      --port=54321          Port to listen on.

## Usage

    HTTP call on / and get JSON with:

    {
      "Timestamp": <string> timestamp of the GPS data in RCF 3339,
      "Longitude": <integer> longitude in decimal degrees,
      "Latitude": <integer> latitude in decimal degrees,
      "LongitudeGPS": <string> longitude in GSP/NMEA coordinates,
      "LatitudeGPS": <string> latitude in GSP/NMEA coordinates,
      "LongitudeDMS": <string> longitude in degrees, minutes, seconds,
      "LatitudeDMS": <string> latitude in degrees, minutes, seconds,
      "Altitude": <integer> altitude in meters,
      "Satellites": <integer> number of satellites,
      "Age": <integer> nanoseconds since last update of these data,
    }

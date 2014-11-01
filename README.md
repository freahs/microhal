microhal
========

A chatbot inspired by MegaHal

Usage
-----

Initialize either a new Microhal instance with ```microhal.NewMicrohal(name, order)``` or load a database with ```microhal.LoadMicrohal(name)```.

To start the bot use ```microhal.Start(saveInterval, maxLength)``` which returns two channels, one for input and one for output. The bot will respond on the output channel to strings sent to the input channel.

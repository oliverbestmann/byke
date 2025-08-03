package audio

type channelStream[S Sample] struct {
	AudioStream[S]
	channel int
}

func SelectChannel[S Sample](stream AudioStream[S], channel int) AudioStream[S] {
	return channelStream[S]{
		AudioStream: stream,
		channel:     channel,
	}
}

func (c channelStream[S]) Read(samples []S) (int, error) {
	config := c.Config()

	n, err := c.AudioStream.Read(samples)

	if n > 0 {
		for idx := range samples[:n] {
			// clear all non selected channels
			if idx%config.Channels != c.channel {
				samples[idx] = 0
			}
		}
	}

	return n, err
}

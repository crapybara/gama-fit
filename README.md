# GamaFit

## What is GamaFit?

GamaFit is a self-care and fitness hub designed to help users improve their health, productivity, and consistency. The goal is simple: help users reach their goals while providing useful tools and data along the way.

Rather than acting as a strict coach, GamaFit treats users as capable adults and avoids unnecessary restrictions, notifications, or artificial engagement systems.

## Why This Project?

I always wanted an application that could support multiple areas of self-improvement in one place. Most apps focus on a single area, such as fitness, study, or nutrition.

GamaFit was built to be the tool I wanted for myself. If it helps me stay consistent and organized, hopefully it can help others as well.

## Features

### Fitness Tracking

Track the three major pillars of physical health:

* Sleep
* Nutrition
* Workouts

### Analytics & Progress Tracking

Monitor your progress with detailed statistics and historical data:

* Body weight tracking
* Lift progression and workout history
* Muscle focus analysis
* Training volume analysis
* Pomodoro and study statistics

### Additional Features

* Habit and progress logging
* Mark logger
* Gama CLI companion tool
* Multiple themes
* Database import/export
* Multi-user support

## Philosophy

GamaFit is designed to support users without becoming intrusive.

The application:

* Does not impose arbitrary rules
* Does not force engagement loops
* Does not treat users like children

The goal is to provide useful tools and let users decide how to use them.

## Notes

### Study Mode Customization

Study mode supports custom assets.

#### Background Videos

Place videos in:

```text
/external/assets/videos
```

#### Music

Add music files to the music directory used by the application.

#### Quotes

Custom quotes can be added in:

```text
/quotes
```

### Rebuilding

After adding assets:

```bash
docker compose up --build
```

or rebuild and restart the Go server manually.

# Screenshots

## Dashboard

![Dashboard](https://files.catbox.moe/hbzmga.png)

## Fitness Analytics

![Fitness Analytics](https://files.catbox.moe/7ubfnm.png)

## Workout Tracking

![Workout Tracking](https://files.catbox.moe/4hkg7l.png)

## Muscle Analysis

![Muscle Analysis](https://files.catbox.moe/ifz22h.png)

## Nutrition Tracking

![Nutrition Tracking](https://files.catbox.moe/c1ndsu.png)

## Study Mode

![Study Mode](https://files.catbox.moe/m4w5v6.png)

## Pomodoro Statistics

![Pomodoro Statistics](https://files.catbox.moe/adyro7.png)

## Additional Features

![Additional Features](https://files.catbox.moe/tp507e.png)

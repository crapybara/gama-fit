# Gama Fitness Analytics & Calculations Guide

This document outlines how various metrics and statistics are calculated across the Gama Fitness platform.

## 1. Body Composition & Health Metrics

### Body Fat Percentage (%)
The app uses the **U.S. Navy Method** if neck and waist measurements are provided:
- **Males:** `86.010 * log10(belly - neck) - 70.041 * log10(height) + 36.76`
- **Females:** `163.205 * log10(belly - neck) - 97.684 * log10(height) - 78.387`
*(Note: A hard floor of 3% is enforced to prevent unrealistic numbers).*

### Lean Body Mass (LBM)
If body fat is successfully calculated:
`LBM = Weight * (1 - BodyFat / 100)`

If body fat cannot be calculated (measurements missing), the app falls back to the **Janmahasatian Formula** for LBM:
- **Males:** `(9270.0 * Weight) / (6680.0 + 216.0 * BMI)`
- **Females:** `(9270.0 * Weight) / (8780.0 + 244.0 * BMI)`
*(The Body Fat percentage is then derived from this LBM).*

### Fat-Free Mass Index (FFMI)
`FFMI = LBM / (Height in meters)^2`

## 2. Analytics Dashboards & Time Ranges

The platform relies heavily on **Rolling Windows** to provide accurate tracking.

- **Weight Trajectory & Trends (Nutrition Page):** Uses a **90-day rolling window**. It performs a linear regression (`Slope = Σ((x - mx)(y - my)) / Σ((x - mx)^2)`) to determine the exact trend (kg/week) and ignore daily water weight fluctuations.
- **Top Stats (Total Overview):** Strictly uses a **7-day rolling window**.
  - Current Week Volume: Past 7 days.
  - Previous Week Volume: Day 14 to Day 8 ago.
- **Chart Renderings (1m, 3m, 6m, YTD):** The charts dynamically adjust to the selected dropdown range, plotting either `avg_weight` or `max_weight` per logged day based on the user's toggle preference.

## 3. Total Overview: Goal Progress Calculation

The "Total Overview" dashboard compares the user's logged output against their set weekly targets:
- **Calories Progress (%):** `(Logged Calories for Week / Target Calories for Week) * 100`
- **Protein Progress (%):** `(Logged Protein for Week / Target Protein for Week) * 100`
- **Volume Progress (%):** `(Logged Volume / Planned Workout Volume) * 100`

If the user exceeds 100% of their Volume goal, the UI handles the overflow seamlessly and displays the absolute Extra Volume achieved (`LoggedVolume - PlannedVolume`).

## 4. Exercise Progression Data Points

When rendering the "Exercise Progression" chart, the backend calculates both averages and maximums for each logged day simultaneously:
- **Avg Mode:** Displays `AVG(weight)` and `AVG(reps)` for all sets performed on that date.
- **Top Set Mode:** Displays `MAX(weight)` and the corresponding `reps` of that maximum set.

*(Both are queried efficiently via PostgreSQL Common Table Expressions (CTEs) combining `GROUP BY` and `DISTINCT ON` operations).*

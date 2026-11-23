<script setup lang="ts">
// Claude is working: four balls in the four primary colours (red, yellow,
// green, blue: the theme's status colours), each colour once, in a random
// order. Three animations take turns: Nawsome's rings
// (uiverse.io/Nawsome/young-goat-78), mobinkakei's bouncing balls
// (uiverse.io/mobinkakei/grumpy-turtle-41) and adamgiebl's pulsing dots
// (uiverse.io/adamgiebl/thin-lionfish-5). All have the same balls in the same
// places: in a row across the middle of a box 8 balls wide.
import { onBeforeUnmount, ref } from 'vue'

defineProps<{ label?: string }>()

const colours = ['var(--approval)', 'var(--waiting)', 'var(--done)', 'var(--running)']
for (let i = colours.length - 1; i > 0; i--) {
  const j = Math.floor(Math.random() * (i + 1))
  ;[colours[i], colours[j]] = [colours[j], colours[i]]
}
// The first at random, then one of the two others at random: each a third
// of the time, never twice in a row. Each runs 6s (the pulsing dots 6.6s, the last one starting 0.6s late; the rings 7.8s: three turns
// of 2.6s, which then rest in the row); the bouncing balls take 0.45s more: the last
// one starts 0.3s after the first, and each settles after its last landing
// (0.15s).
const kinds = ['rings', 'bounce', 'pulse'] as const
const kind = ref(Math.floor(Math.random() * 3))
let timer = 0
function next() {
  timer = window.setTimeout(() => {
    kind.value = (kind.value + 1 + Math.floor(Math.random() * 2)) % 3
    next()
  }, kinds[kind.value] === 'bounce' ? 6450 : kinds[kind.value] === 'rings' ? 7800 : 6600)
}
next()
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <span class="working" role="status" :aria-label="label || 'Claude is working'">
    <span v-if="kinds[kind] === 'bounce'" class="bounce" aria-hidden="true">
      <span v-for="(c, i) in colours" :key="i" class="hop" :style="{ '--c': c, '--i': i }"></span>
    </span>
    <span v-else-if="kinds[kind] === 'pulse'" class="pulse" aria-hidden="true">
      <span v-for="(c, i) in colours" :key="i" class="blip" :style="{ '--c': c, '--i': i }"></span>
    </span>
    <svg v-else class="rings" viewBox="0 0 240 240" aria-hidden="true">
      <circle class="ring a" cx="120" cy="120" r="105" :style="{ stroke: colours[0] }" stroke-dasharray="0 660" stroke-dashoffset="-330" />
      <circle class="ring b" cx="120" cy="120" r="35" :style="{ stroke: colours[1] }" stroke-dasharray="0 220" stroke-dashoffset="-110" />
      <circle class="ring c" cx="85" cy="120" r="70" :style="{ stroke: colours[2] }" stroke-dasharray="0 440" />
      <circle class="ring d" cx="155" cy="120" r="70" :style="{ stroke: colours[3] }" stroke-dasharray="0 440" />
    </svg>
    <span v-if="label" class="label">{{ label }}</span>
  </span>
</template>

<style scoped>
.working {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  --ball: 5px; /* the balls of all three */
}
.label {
  font-size: var(--fs-12);
  color: var(--muted);
}

/* The rings: a stroke of 30 in a 240 box is one ball, so the box is 8 balls
   wide. The original swells the balls from 20 to 30 as they move; they keep
   30 here, as round as the bouncing ones. At rest, they lie in a row
   across the middle, 70 apart: the bouncing balls land there too. */
.rings {
  flex: none;
  width: calc(var(--ball) * 8);
  height: calc(var(--ball) * 8);
}
.ring {
  fill: none;
  stroke-width: 30;
  stroke-linecap: round;
  animation: ringA 2.6s linear infinite;
}
.ring.b {
  animation-name: ringB;
}
.ring.c {
  animation-name: ringC;
}
.ring.d {
  animation-name: ringD;
}
@keyframes ringA {
  from,
  4% {
    stroke-dasharray: 0 660;
    stroke-dashoffset: -330;
  }
  12% {
    stroke-dasharray: 60 600;
    stroke-dashoffset: -335;
  }
  32% {
    stroke-dasharray: 60 600;
    stroke-dashoffset: -595;
  }
  40%,
  54% {
    stroke-dasharray: 0 660;
    stroke-dashoffset: -660;
  }
  62% {
    stroke-dasharray: 60 600;
    stroke-dashoffset: -665;
  }
  82% {
    stroke-dasharray: 60 600;
    stroke-dashoffset: -925;
  }
  90%,
  to {
    stroke-dasharray: 0 660;
    stroke-dashoffset: -990;
  }
}
@keyframes ringB {
  from,
  12% {
    stroke-dasharray: 0 220;
    stroke-dashoffset: -110;
  }
  20% {
    stroke-dasharray: 20 200;
    stroke-dashoffset: -115;
  }
  40% {
    stroke-dasharray: 20 200;
    stroke-dashoffset: -195;
  }
  48%,
  62% {
    stroke-dasharray: 0 220;
    stroke-dashoffset: -220;
  }
  70% {
    stroke-dasharray: 20 200;
    stroke-dashoffset: -225;
  }
  90% {
    stroke-dasharray: 20 200;
    stroke-dashoffset: -305;
  }
  98%,
  to {
    stroke-dasharray: 0 220;
    stroke-dashoffset: -330;
  }
}
@keyframes ringC {
  from {
    stroke-dasharray: 0 440;
    stroke-dashoffset: 0;
  }
  8% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -5;
  }
  28% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -175;
  }
  36%,
  58% {
    stroke-dasharray: 0 440;
    stroke-dashoffset: -220;
  }
  66% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -225;
  }
  86% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -395;
  }
  94%,
  to {
    stroke-dasharray: 0 440;
    stroke-dashoffset: -440;
  }
}
@keyframes ringD {
  from,
  8% {
    stroke-dasharray: 0 440;
    stroke-dashoffset: 0;
  }
  16% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -5;
  }
  36% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -175;
  }
  44%,
  50% {
    stroke-dasharray: 0 440;
    stroke-dashoffset: -220;
  }
  58% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -225;
  }
  78% {
    stroke-dasharray: 40 400;
    stroke-dashoffset: -395;
  }
  86%,
  to {
    stroke-dasharray: 0 440;
    stroke-dashoffset: -440;
  }
}

/* The bouncing balls, without the original's shadows: four, landing where
   the rings' balls rest (7/3 of a ball apart, across the middle of the same
   box), bouncing three balls high and squashing as they land. */
.bounce {
  flex: none;
  position: relative;
  width: calc(var(--ball) * 8);
  height: calc(var(--ball) * 8);
}
.hop {
  position: absolute;
  top: calc(var(--ball) * 3.5); /* at rest, centred on the middle line */
  left: calc(var(--ball) * 7 / 3 * var(--i));
  width: var(--ball);
  height: var(--ball);
  border-radius: 50%;
  background: var(--c);
  /* six bounces (twelve half bounces), then, landed squashed, it rounds
     back up on the line (settle) instead of jumping back to round */
  animation:
    hop 0.5s calc(var(--i) * 0.1s) alternate 12 ease,
    settle 0.15s ease-out calc(var(--i) * 0.1s + 6s) forwards;
}
@keyframes settle {
  from {
    top: calc(var(--ball) * 4.25);
    height: calc(var(--ball) / 4);
    border-radius: 50px 50px 25px 25px;
    transform: scaleX(1.7);
  }
  to {
    top: calc(var(--ball) * 3.5);
    height: var(--ball);
    border-radius: 50%;
    transform: scaleX(1);
  }
}
@keyframes hop {
  0% {
    /* landed and squashed: its bottom where a resting ball's is */
    top: calc(var(--ball) * 4.25);
    height: calc(var(--ball) / 4);
    border-radius: 50px 50px 25px 25px;
    transform: scaleX(1.7);
  }
  40% {
    height: var(--ball);
    border-radius: 50%;
    transform: scaleX(1);
  }
  100% {
    top: calc(var(--ball) * 1.25);
  }
}

/* The pulsing dots: four, where the rings' balls rest, swelling in turn,
   without the original's fading halo; each keeps its colour. */
.pulse {
  flex: none;
  position: relative;
  width: calc(var(--ball) * 8);
  height: calc(var(--ball) * 8);
}
.blip {
  position: absolute;
  top: calc(var(--ball) * 3.5);
  left: calc(var(--ball) * 7 / 3 * var(--i));
  width: var(--ball);
  height: var(--ball);
  border-radius: 50%;
  background: var(--c);
  animation: blip 1.5s calc(var(--i) * 0.2s) 4 ease-in-out;
}
@keyframes blip {
  0%,
  100% {
    transform: scale(1);
  }
  25% {
    transform: scale(0.45);
  }
  70% {
    transform: scale(1.6);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ring,
  .blip,
  .hop {
    animation-duration: 4s;
  }
}
</style>

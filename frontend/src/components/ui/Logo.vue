<script setup lang="ts">
// jockey's logo: the jockey helmet of build/appicon.png, without its tile.
import logo from '../../assets/logo.png'

withDefaults(defineProps<{ size?: number }>(), { size: 96 })
</script>

<template>
  <img :src="logo" :width="size" :height="size" alt="" aria-hidden="true" draggable="false" />
</template>

<template>
	<span @mouseenter="show" @mouseleave="visible = false">
		<slot />
		<div ref="tooltipTrigger" :style="{ position: 'absolute' }">
			<div ref="tooltip" class="tooltip" :visible="visible">
				<span>{{ text }}</span>
			</div>
		</div>
	</span>
</template>

<script setup lang="ts">
import type { Placement } from "@popperjs/core";
import { Instance, VirtualElement, createPopper } from "@popperjs/core";
import { PropType, ref } from "vue";

const props = defineProps({
	text: {
		type: String,
		default: "",
	},
	offset: {
		type: Object as PropType<number[]>,
	},
	position: {
		type: String as PropType<Placement>,
		default: "auto",
	},
});

const visible = ref(false);
const tooltip = ref<HTMLElement | null>(null);
const tooltipTrigger = ref<HTMLElement | null>(null);

let popper: Instance;
const show = (ev: MouseEvent) => {
	if (popper) {
		popper.destroy();
	}
	if (!tooltip.value) {
		return;
	}
	visible.value = true;

	const x = ev.clientX;
	const y = ev.clientY;

	const trigger = {
		getBoundingClientRect: () => ({
			width: 0,
			height: 0,
			top: y,
			right: x,
			bottom: y,
			left: x,
		}),
	} as VirtualElement;

	popper = createPopper(trigger, tooltip.value as HTMLElement, {
		placement: props.position,
		modifiers: [
			{
				name: "offset",
				options: {
					offset: props.offset,
				},
			},
		],
	});
};
</script>

<style lang="scss" scoped>
@import "@scss/themes.scss";

.tooltip {
	z-index: 100;
	pointer-events: none;
	width: max-content;
	max-width: 24em;
	font-size: 1rem;

	&[visible="false"] {
		display: none;
	}

	> span {
		padding: 0.5em;
		@include themify() {
			background-color: lighten(themed("backgroundColor"), 2);
			box-shadow: 0.016em 0.016em 0.2em themed("color");
		}
	}
}
</style>

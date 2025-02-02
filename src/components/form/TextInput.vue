<template>
	<div class="text-input">
		<input :value="modelValue" :empty="!modelValue?.length" @input="onInput" @blur="$emit('blur')" />
		<label>
			<span> {{ label }} </span>
		</label>
	</div>
</template>

<script setup lang="ts">
import { PropType } from "vue";

defineProps({
	label: String,
	modelValue: String,
	icon: {
		type: Object as PropType<[string, string]>,
	},
	appearance: {
		type: String as PropType<"flat" | "outline">,
		default: "flat",
	},
});

const emit = defineEmits(["update:modelValue", "blur"]);
const onInput = (event: Event) => emit("update:modelValue", (event.target as HTMLInputElement).value);
</script>

<style lang="scss">
@import "@scss/themes.scss";

.text-input {
	position: relative;

	@include themify() {
		input {
			background-color: lighten(themed("backgroundColor"), 4);
			border-color: mix(themed("backgroundColor"), themed("color"), 85);
		}
		input:focus {
			border-color: mix(themed("primary"), themed("backgroundColor"), 40);
		}
	}

	label {
		pointer-events: none;
		position: absolute;
		top: 0.6em;
		left: 0;
		transition: transform 200ms ease;
		margin-left: 0.5em;
	}

	input {
		transition: border-color 140ms ease-in-out;
		background-color: transparent;
		color: inherit;
		border-width: 0.05em;
		border-style: solid;
		border-color: transparent;
		border-radius: 0.25em;
		padding: 0.5em;
		font-size: 1em;
		font-weight: 500;
		width: 100%;
	}

	input[empty="false"],
	input:focus {
		outline: unset;
		& ~ label {
			font-weight: 600;
			transform: translateY(-1.35em) translateX(-0.25em) scale(0.8);
		}
	}
}
</style>

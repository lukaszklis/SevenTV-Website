<template>
	<main ref="page" class="emote-page">
		<template v-if="partial || (emote && !loading)">
			<!-- Heading Bar | Emote Title / Author -->
			<section class="heading-bar">
				<div v-if="emote?.owner" class="emote-author">
					<span>{{ t("emote.author") }}</span>
					<UserTag scale="1.5em" text-scale="1.3rem" :user="emote?.owner" :clickable="true" />
				</div>
				<div v-if="emote" class="emote-name">
					<p>{{ emote.name }}</p>
					<span v-if="actor.defaultEmoteSetID && customName && customName !== emote.name" class="set-renamed">
						as {{ customName }}
					</span>
				</div>
				<!--
				<div class="creation-date">
					<p>{{ t("emote.created_at") }}</p>
					<span> {{ createdAt }} </span>
				</div>
				-->

				<div class="format-selector-outer">
					<div class="format-selector">
						<LogoWEBP
							:selected="selectedFormat === Common.Image.Format.WEBP"
							class="format-button"
							@click="selectedFormat = Common.Image.Format.WEBP"
						/>
						<LogoAVIF
							:selected="selectedFormat === Common.Image.Format.AVIF"
							class="format-button"
							@click="selectedFormat = Common.Image.Format.AVIF"
						/>
					</div>
				</div>
			</section>

			<!-- Preview Block | Sizes display -->
			<section v-if="preview.images.size > 0 && !isProcessing && preview.loaded" class="preview-block">
				<div
					v-for="(im, index) in preview.images"
					:key="im.getAttribute('filename') ?? ''"
					class="preview-size"
					:class="{ 'is-large': index >= 3 }"
				>
					<img :src="im.src" />
				</div>
			</section>
			<section v-else-if="isProcessing" class="preview-block is-loading">
				<span class="emote-is-processing">Processing Emote - this may take some time.</span>
			</section>
			<section v-else-if="preview.errors < 4" class="preview-block is-loading">
				<span>Loading previews... ({{ preview.count + 1 }}/{{ preview.images?.size }})</span>
			</section>
			<section v-else class="preview-block is-loading">
				<span :style="{ color: 'red' }">Failed to load preview</span>
			</section>

			<!-- Interactions: Actions, Versions & Comments -->
			<section class="interactive-block">
				<div class="emote-interactions">
					<EmoteInteractions :emote="emote" />
				</div>
			</section>

			<div class="level-separation" />
			<section class="informative-block">
				<div section="versioning">
					<div class="section-head">
						<h3>{{ t("emote.versions") }}</h3>
					</div>
					<div class="section-content">
						<div v-if="emote && emote.versions?.length">
							<EmoteVersions :emote="emote" />
						</div>
					</div>
				</div>

				<div v-if="channels" section="channels">
					<div class="section-head">
						<h3>{{ t("emote.channels") }} ({{ channels.total }})</h3>
					</div>
					<div class="section-content">
						<div v-for="u in channels?.items" :key="u.id" class="channel-card-wrapper" :ok="!!u.id">
							<router-link
								:to="
									u.id ? { name: 'User', params: { userID: u.id, userData: JSON.stringify(u) } } : ''
								"
								class="unstyled-link"
								draggable="false"
							>
								<div
									v-wave
									class="channel-card"
									:style="{
										backgroundColor: u.tag_color ? ConvertIntColorToHex(u.tag_color, 0.075) : '',
									}"
								>
									<div class="user-picture">
										<UserTag :user="u" text-scale="0" scale="2.75em" />
									</div>
									<span class="nametag-only">
										<UserTag :user="u" text-scale="0.85em" :hide-avatar="true" />
									</span>
								</div>
							</router-link>
						</div>
					</div>
				</div>

				<div section="comments">
					<div class="section-head">
						<h3>{{ t("common.activity") }}</h3>
					</div>
					<div class="section-content">
						<div class="comment-list">
							<EmoteActivity />
						</div>
					</div>
				</div>
			</section>

			<!-- Scroll section: Statistics -->
			<EmoteStats :emote-i-d="emoteID" />
		</template>
		<template v-else-if="loading">Loading...</template>
		<template v-else>
			<div class="emote-unknown">
				<NotFoundPage />
			</div>
		</template>
	</main>
</template>

<script setup lang="ts">
import { Emote } from "@structures/Emote";
import { computed, onUnmounted, ref, watch } from "vue";
import { useQuery, useSubscription } from "@vue/apollo-composable";
import { OperationVariables } from "@apollo/client/core";
import { GetEmoteChannels, GetEmote, WatchEmote } from "@gql/emotes/emote";
import { ConvertIntColorToHex } from "@structures/util/Color";
import { Common } from "@structures/Common";
import { ApplyMutation } from "@structures/Update";
import { useActorStore } from "@store/actor";
import { useHead } from "@vueuse/head";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import UserTag from "@components/utility/UserTag.vue";
import NotFoundPage from "@views/404.vue";
import EmoteInteractions from "@views/EmotePage/EmoteInteractions.vue";
import EmoteVersions from "@views/EmotePage/EmoteVersions.vue";
import EmoteActivity from "./EmoteActivity.vue";
import LogoAVIF from "@components/base/LogoAVIF.vue";
import LogoWEBP from "@components/base/LogoWEBP.vue";
import EmoteStats from "./EmoteStats.vue";

const { t } = useI18n();

const props = defineProps({
	emoteID: String,
	emoteData: {
		required: false,
		type: String,
	},
	headingOnly: Boolean,
});

const actor = useActorStore();
const emoteID = ref(props.emoteID ?? "");
const emote = ref((props.emoteData ? JSON.parse(props.emoteData) : null) as Emote | null);
const title = computed(() =>
	"".concat(
		emote.value !== null ? emote.value.name : "Emote",
		emote.value?.owner ? ` by ${emote.value.owner.display_name}` : "",
		" - 7TV",
	),
);
useHead({ title });

const isProcessing = computed(
	() => emote.value?.lifecycle === Emote.Lifecycle.PENDING || emote.value?.lifecycle === Emote.Lifecycle.PROCESSING,
);
/** Whether or not the page was initiated with partial emote data  */
const partial = emote.value !== null;

// Fetch emote
const { onResult, loading, stop, refetch } = useQuery<GetEmote>(GetEmote, { id: props.emoteID });
onResult((res) => {
	if (!res.data) {
		return;
	}
	emote.value = res.data.emote;
	defineLinks(selectedFormat.value);
});

// Watch emote
const {
	onResult: onEmoteUpdate,
	restart: restartSub,
	variables: subVariables,
} = useSubscription<GetEmote>(WatchEmote, { id: emoteID.value });
onEmoteUpdate((res) => {
	if (!res.data || !emote.value) {
		return;
	}

	for (const k of Object.keys(res.data.emote)) {
		ApplyMutation(emote.value, {
			action: "set",
			field: k,
			value: JSON.stringify(res.data.emote[k as keyof Emote]),
		});
	}
	defineLinks(selectedFormat.value);
});

// Fetch channels
const { result: getChannels, refetch: refetchChannels } = useQuery<GetEmote>(GetEmoteChannels, {
	id: props.emoteID,
	page: 1,
	limit: 50,
});
const channels = computed<Emote.UserList>(
	() =>
		(preview.value.loaded ? getChannels.value?.emote.channels : null) ?? {
			total: 0,
			items: Array(50).fill({ id: null }),
		},
);

// Handle route changes
const route = useRoute();
watch(route, () => {
	if (route.name !== "Emote") {
		return;
	}
	emoteID.value = String(route.params.emoteID);
	refetch({ id: emoteID.value });
	refetchChannels({ id: emoteID.value, page: 1, limit: 50 });
	(subVariables.value as OperationVariables).id = emoteID.value;
	restartSub();
});

// Format selection
const selectedFormat = ref<Common.Image.Format>(Common.Image.Format.WEBP);

// Preload preview images
const currentVersion = computed(
	() => emote.value?.versions?.filter((ver) => emote.value && ver.id === emote.value.id)[0],
);
const preview = ref({
	loaded: false,
	count: 0,
	errors: 0,
	images: new Set<HTMLImageElement>(),
});
const defineLinks = (format: Common.Image.Format) => {
	let loaded = 0;

	preview.value.images.clear();
	preview.value.count = 0;
	preview.value.errors = 0;

	const imgs =
		currentVersion.value?.images.filter((im) => im.format === format).sort((a, b) => a.width - b.width) ??
		emote.value?.images ??
		[];
	if (imgs.length < 4) {
		preview.value.errors = 4;
	}
	for (const im of imgs) {
		const w = im.width;
		const h = im.height;
		const img = new Image(w, h);
		preview.value.images.add(img);
		img.src = im.url;
		img.setAttribute("filename", im.name);

		const listener: (this: HTMLImageElement, ev: Event) => void = function () {
			loaded++;
			preview.value.count = loaded;

			if (loaded >= 4) {
				preview.value.loaded = true;
				img.removeEventListener("load", listener);
			}
		};
		img.addEventListener("load", listener);
		img.onerror = () => {
			preview.value.errors++;
		};
	}
};
if (partial) {
	defineLinks(Common.Image.Format.WEBP);
}
watch(selectedFormat, (format) => defineLinks(format));

const page = ref<HTMLDivElement | null>(null);
onUnmounted(() => {
	// Halt query
	stop();
	emote.value = null;
});

// const hasEmote = computed(() => (emote.value ? store.activeEmotes.includes(emote.value?.id) : false));
// const createdAt = computed(() => formatDate("MMMM d, y")(new Date(emote.value?.created_at ?? 0)));
const customName = computed(() =>
	actor.defaultEmoteSetID && emote.value
		? actor.getActiveEmoteInSet(actor.defaultEmoteSetID, emote.value?.id)?.name ?? ""
		: "",
);
</script>

<style lang="scss">
@import "@scss/emote-page/emote-page.scss";
</style>

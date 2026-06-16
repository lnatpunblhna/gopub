<template>
    <section class="body-warp">
        <router-view name="fullView" v-slot="{ Component }">
            <transition name="fade" mode="out-in">
                <component :is="Component" v-if="Component" style="z-index: 200"></component>
            </transition>
        </router-view>
        <router-view name="menuView" v-slot="{ Component }">
            <transition name="fade" mode="out-in">
                <component :is="Component" v-if="Component" style="z-index: 150"></component>
            </transition>
        </router-view>
        <main-content style="z-index: 100">
            <router-view v-slot="{ Component }">
                <transition name="fade" mode="out-in">
                    <component :is="Component" v-if="Component"></component>
                </transition>
            </router-view>
        </main-content>
    </section>
</template>

<script>
    import {mainContent} from 'components'

    export default {
        name: 'app',
        components: {
            mainContent
        }
    }
</script>

<style lang="scss">
    @import './assets/scss/main';

    .fade-enter-active,
    .fade-leave-active {
        transition: all .2s ease;
    }

    .fade-enter-from,
    .fade-leave-to {
        opacity: 0;
    }
</style>

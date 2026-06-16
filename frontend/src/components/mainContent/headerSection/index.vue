<template>
    <div class="header-section">
        <div class="header-title">
            <strong>{{ $route.meta.title || '工作台' }}</strong>
            <span>gopub console</span>
        </div>
        <menu-right></menu-right>
    </div>
</template>
<script type="text/javascript">
    import menuRight from './menuRight/index.vue'

    export default{
        components: {
            menuRight
        }
    }
</script>

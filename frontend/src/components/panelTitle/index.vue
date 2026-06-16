<template>
    <div class="panel-title">
        <span v-if="title" class="panel-title-text" v-text="title"></span>
        <div class="panel-title-actions">
            <slot></slot>
        </div>
    </div>
</template>
<script type="text/javascript">
    export default{
        props: {
            title: {
                type: String
            }
        }
    }
</script>

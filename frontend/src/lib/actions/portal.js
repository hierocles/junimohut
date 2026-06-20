/** Move a node to document.body so position:fixed uses the viewport. */
export function portal(node) {
    document.body.appendChild(node);
    return {
        destroy() {
            node.remove();
        },
    };
}
//# sourceMappingURL=portal.js.map
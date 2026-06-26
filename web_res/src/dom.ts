// getElement looks up an element by id and throws a clear error if it is
// missing, so callers get type-safe access without non-null assertions.
export function getElement<T extends HTMLElement>(id: string): T {
    const el = document.getElementById(id);
    if (el === null) {
        throw new Error(`missing element #${id}`);
    }
    return el as T;
}

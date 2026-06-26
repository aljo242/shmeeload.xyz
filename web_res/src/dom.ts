// getElement looks up an element by id and throws a clear error if it is
// missing, avoiding non-null assertions at call sites. Note: the generic T is an
// unchecked cast (the caller asserts the element type), so only use it where the
// id reliably maps to that element type.
export function getElement<T extends HTMLElement>(id: string): T {
    const el = document.getElementById(id);
    if (el === null) {
        throw new Error(`missing element #${id}`);
    }
    return el as T;
}

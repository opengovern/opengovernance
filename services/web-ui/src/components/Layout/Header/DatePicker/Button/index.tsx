import { useRef } from 'react'
import {
    useButton,
    useFocusRing,
    mergeProps,
    AriaButtonProps,
} from 'react-aria'

export function CalendarButton(props: AriaButtonProps) {
    const ref = useRef(null)
    const { buttonProps } = useButton(props, ref)
    const { focusProps, isFocusVisible } = useFocusRing()
    return (
        <button
            type="button"
            {...mergeProps(buttonProps, focusProps)}
            ref={ref}
            className={`p-2 rounded-full ${
                props?.isDisabled ? 'text-gray-400' : ''
            } ${
                !props?.isDisabled
                    ? 'hover:bg-openg-100 active:bg-openg-200'
                    : ''
            } outline-none ${
                isFocusVisible ? 'ring-2 ring-offset-2 ring-openg-600' : ''
            }`}
        >
            {props?.children}
        </button>
    )
}

export function FieldButton(props: AriaButtonProps & { isPressed?: boolean }) {
    const ref = useRef(null)
    const { buttonProps, isPressed } = useButton(props, ref)
    return (
        <button
            type="button"
            {...buttonProps}
            ref={ref}
            className={`px-2 -ml-px border dark:border-gray-700 transition-colors rounded-r-lg group-focus-within:border-openg-600 group-focus-within:group-hover:border-blue-600 outline-none ${
                isPressed || props?.isPressed
                    ? 'bg-gray-200 border-gray-400 dark:bg-gray-700'
                    : 'bg-gray-50 border-gray-300 dark:bg-gray-700 group-hover:border-gray-400'
            }`}
        >
            {props?.children}
        </button>
    )
}

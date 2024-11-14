import { DismissButton, Overlay, usePopover } from '@react-aria/overlays'
import { useRef } from 'react'

export function Popover(props: any) {
    const ref = useRef(null)
    const { state, children } = props

    const { popoverProps, underlayProps } = usePopover(
        {
            ...props,
            popoverRef: ref,
        },
        state
    )

    return (
        <Overlay>
            <div {...underlayProps} className="fixed inset-0" />
            <div
                {...popoverProps}
                ref={ref}
                className="bg-white dark:bg-openg-950 border border-gray-300 dark:border-gray-700 rounded-xl shadow-lg mt-2 p-4 z-10"
            >
                <DismissButton onDismiss={state.close} />
                {children}
                <DismissButton onDismiss={state.close} />
            </div>
        </Overlay>
    )
}

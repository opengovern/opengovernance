import { useDialog } from 'react-aria'
import React from 'react'

export function Dialog({ children, ...props }: { children: React.ReactNode }) {
    const ref = React.useRef(null)
    const { dialogProps } = useDialog(props, ref)

    return (
        <div {...dialogProps} ref={ref}>
            {children}
        </div>
    )
}

import { useRef } from 'react'
import { useCalendarCell } from 'react-aria'
import { AriaCalendarCellProps } from '@react-aria/calendar'
import { CalendarState } from 'react-stately'

export function CalendarCell({
    state,
    date,
}: {
    state: CalendarState
    date: AriaCalendarCellProps['date']
}) {
    const ref = useRef(null)
    const {
        cellProps,
        buttonProps,
        isSelected,
        isOutsideVisibleRange,
        isDisabled,
        formattedDate,
        isInvalid,
    } = useCalendarCell({ date }, state, ref)
    // console.log(state)

    return (
        <td {...cellProps} className="py-0.5">
            <div
                {...buttonProps}
                ref={ref}
                hidden={isOutsideVisibleRange}
                className={`w-10 h-10 outline-none group ${
                    isSelected ? 'bg-openg-700 text-gray-50 rounded-md' : ''
                } ${isDisabled ? 'disabled' : ''}`}
            >
                <div
                    className={`w-full h-full rounded-md flex items-center justify-center ${
                        isDisabled && !isInvalid ? 'text-gray-400' : ''
                    } ${
                        // Hover state for non-selected cells.
                        !isSelected && !isDisabled ? 'hover:bg-openg-100' : ''
                    } cursor-default`}
                >
                    {formattedDate}
                </div>
            </div>
        </td>
    )
}

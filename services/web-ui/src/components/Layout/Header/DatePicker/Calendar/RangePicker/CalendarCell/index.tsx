import { useRef } from 'react'
import {
    useCalendarCell,
    useLocale,
    useFocusRing,
    mergeProps,
} from 'react-aria'
import { isSameDay, getDayOfWeek } from '@internationalized/date'
import { AriaCalendarCellProps } from '@react-aria/calendar'
import { RangeCalendarState } from 'react-stately'

export function CalendarCell({
    state,
    date,
}: {
    state: RangeCalendarState
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

    // The start and end date of the selected range will have
    // an emphasized appearance.
    const isSelectionStart = state.highlightedRange
        ? isSameDay(date, state.highlightedRange.start)
        : isSelected
    const isSelectionEnd = state.highlightedRange
        ? isSameDay(date, state.highlightedRange.end)
        : isSelected

    // We add rounded corners on the left for the first day of the month,
    // the first day of each week, and the start date of the selection.
    // We add rounded corners on the right for the last day of the month,
    // the last day of each week, and the end date of the selection.
    const { locale } = useLocale()
    const dayOfWeek = getDayOfWeek(date, locale)
    const isRoundedLeft =
        isSelected && (isSelectionStart || dayOfWeek === 0 || date.day === 1)
    const isRoundedRight =
        isSelected &&
        (isSelectionEnd ||
            dayOfWeek === 6 ||
            date.day === date.calendar.getDaysInMonth(date))

    const { focusProps, isFocusVisible } = useFocusRing()

    if (isSelected) {
        if (isSelectionStart || isSelectionEnd) {
            if (
                isSelected &&
                !isDisabled &&
                !(isSelectionStart || isSelectionEnd)
            ) {
                return (
                    <td
                        {...cellProps}
                        className={`py-0.5 relative ${
                            isFocusVisible ? 'z-10' : 'z-0'
                        }`}
                    >
                        <div
                            {...mergeProps(buttonProps, focusProps)}
                            ref={ref}
                            hidden={isOutsideVisibleRange}
                            className={`w-10 h-10 outline-none group ${
                                isRoundedLeft ? 'rounded-l-md' : ''
                            } ${isRoundedRight ? 'rounded-r-md' : ''} ${
                                isInvalid ? 'bg-red-300' : 'bg-openg-300'
                            } ${isDisabled ? 'disabled' : ''}`}
                        >
                            <div
                                className={`w-full h-full rounded-md flex items-center justify-center ${
                                    isDisabled && !isInvalid
                                        ? 'text-gray-400'
                                        : ''
                                } ${
                                    // Focus ring, visible while the cell has keyboard focus.
                                    isFocusVisible
                                        ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                        : ''
                                } ${
                                    // Darker selection background for the start and end.
                                    isInvalid
                                        ? 'bg-red-600 text-white hover:bg-red-700'
                                        : 'bg-openg-600 text-white hover:bg-openg-700'
                                } ${
                                    // Hover state for cells in the middle of the range.
                                    isInvalid
                                        ? 'hover:bg-red-400'
                                        : 'hover:bg-openg-400'
                                } ${
                                    // Hover state for non-selected cells.
                                    !isSelected && !isDisabled
                                        ? 'hover:bg-openg-100'
                                        : ''
                                } cursor-default`}
                            >
                                {formattedDate}
                            </div>
                        </div>
                    </td>
                )
            }
            return (
                <td
                    {...cellProps}
                    className={`py-0.5 relative ${
                        isFocusVisible ? 'z-10' : 'z-0'
                    }`}
                >
                    <div
                        {...mergeProps(buttonProps, focusProps)}
                        ref={ref}
                        hidden={isOutsideVisibleRange}
                        className={`w-10 h-10 outline-none group ${
                            isRoundedLeft ? 'rounded-l-md' : ''
                        } ${isRoundedRight ? 'rounded-r-md' : ''} ${
                            isInvalid ? 'bg-red-300' : 'bg-openg-300'
                        } ${isDisabled ? 'disabled' : ''}`}
                    >
                        <div
                            className={`w-full h-full rounded-md flex items-center justify-center ${
                                isDisabled && !isInvalid ? 'text-gray-400' : ''
                            } ${
                                // Focus ring, visible while the cell has keyboard focus.
                                isFocusVisible
                                    ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                    : ''
                            } ${
                                // Darker selection background for the start and end.
                                isInvalid
                                    ? 'bg-red-600 text-white hover:bg-red-700'
                                    : 'bg-openg-600 text-white hover:bg-openg-700'
                            } ${
                                // Hover state for cells in the middle of the range.
                                ''
                            } ${
                                // Hover state for non-selected cells.
                                !isSelected && !isDisabled
                                    ? 'hover:bg-openg-100'
                                    : ''
                            } cursor-default`}
                        >
                            {formattedDate}
                        </div>
                    </div>
                </td>
            )
        }
        if (
            isSelected &&
            !isDisabled &&
            !(isSelectionStart || isSelectionEnd)
        ) {
            return (
                <td
                    {...cellProps}
                    className={`py-0.5 relative ${
                        isFocusVisible ? 'z-10' : 'z-0'
                    }`}
                >
                    <div
                        {...mergeProps(buttonProps, focusProps)}
                        ref={ref}
                        hidden={isOutsideVisibleRange}
                        className={`w-10 h-10 outline-none group ${
                            isRoundedLeft ? 'rounded-l-md' : ''
                        } ${isRoundedRight ? 'rounded-r-md' : ''} ${
                            isInvalid ? 'bg-red-300' : 'bg-openg-300'
                        } ${isDisabled ? 'disabled' : ''}`}
                    >
                        <div
                            className={`w-full h-full rounded-md flex items-center justify-center ${
                                isDisabled && !isInvalid ? 'text-gray-400' : ''
                            } ${
                                // Focus ring, visible while the cell has keyboard focus.
                                isFocusVisible
                                    ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                    : ''
                            } ${
                                // Darker selection background for the start and end.
                                ''
                            } ${
                                // Hover state for cells in the middle of the range.
                                isInvalid
                                    ? 'hover:bg-red-400'
                                    : 'hover:bg-openg-400'
                            } ${
                                // Hover state for non-selected cells.
                                !isSelected && !isDisabled
                                    ? 'hover:bg-openg-100'
                                    : ''
                            } cursor-default`}
                        >
                            {formattedDate}
                        </div>
                    </div>
                </td>
            )
        }
        return (
            <td
                {...cellProps}
                className={`py-0.5 relative ${isFocusVisible ? 'z-10' : 'z-0'}`}
            >
                <div
                    {...mergeProps(buttonProps, focusProps)}
                    ref={ref}
                    hidden={isOutsideVisibleRange}
                    className={`w-10 h-10 outline-none group ${
                        isRoundedLeft ? 'rounded-l-md' : ''
                    } ${isRoundedRight ? 'rounded-r-md' : ''} ${
                        isInvalid ? 'bg-red-300' : 'bg-openg-300'
                    } ${isDisabled ? 'disabled' : ''}`}
                >
                    <div
                        className={`w-full h-full rounded-md flex items-center justify-center ${
                            isDisabled && !isInvalid ? 'text-gray-400' : ''
                        } ${
                            // Focus ring, visible while the cell has keyboard focus.
                            isFocusVisible
                                ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                : ''
                        } ${
                            // Darker selection background for the start and end.
                            ''
                        } ${
                            // Hover state for cells in the middle of the range.
                            ''
                        } ${
                            // Hover state for non-selected cells.
                            !isSelected && !isDisabled
                                ? 'hover:bg-openg-100'
                                : ''
                        } cursor-default`}
                    >
                        {formattedDate}
                    </div>
                </div>
            </td>
        )
    }
    if (isSelectionStart || isSelectionEnd) {
        if (
            isSelected &&
            !isDisabled &&
            !(isSelectionStart || isSelectionEnd)
        ) {
            return (
                <td
                    {...cellProps}
                    className={`py-0.5 relative ${
                        isFocusVisible ? 'z-10' : 'z-0'
                    }`}
                >
                    <div
                        {...mergeProps(buttonProps, focusProps)}
                        ref={ref}
                        hidden={isOutsideVisibleRange}
                        className={`w-10 h-10 outline-none group ${
                            isRoundedLeft ? 'rounded-l-md' : ''
                        } ${isRoundedRight ? 'rounded-r-md' : ''} ${''} ${
                            isDisabled ? 'disabled' : ''
                        }`}
                    >
                        <div
                            className={`w-full h-full rounded-md flex items-center justify-center ${
                                isDisabled && !isInvalid ? 'text-gray-400' : ''
                            } ${
                                // Focus ring, visible while the cell has keyboard focus.
                                isFocusVisible
                                    ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                    : ''
                            } ${
                                // Darker selection background for the start and end.
                                isInvalid
                                    ? 'bg-red-600 text-white hover:bg-red-700'
                                    : 'bg-openg-600 text-white hover:bg-openg-700'
                            } ${
                                // Hover state for cells in the middle of the range.
                                isInvalid
                                    ? 'hover:bg-red-400'
                                    : 'hover:bg-openg-400'
                            } ${
                                // Hover state for non-selected cells.
                                !isSelected && !isDisabled
                                    ? 'hover:bg-openg-100'
                                    : ''
                            } cursor-default`}
                        >
                            {formattedDate}
                        </div>
                    </div>
                </td>
            )
        }
        return (
            <td
                {...cellProps}
                className={`py-0.5 relative ${isFocusVisible ? 'z-10' : 'z-0'}`}
            >
                <div
                    {...mergeProps(buttonProps, focusProps)}
                    ref={ref}
                    hidden={isOutsideVisibleRange}
                    className={`w-10 h-10 outline-none group ${
                        isRoundedLeft ? 'rounded-l-md' : ''
                    } ${isRoundedRight ? 'rounded-r-md' : ''} ${''} ${
                        isDisabled ? 'disabled' : ''
                    }`}
                >
                    <div
                        className={`w-full h-full rounded-md flex items-center justify-center ${
                            isDisabled && !isInvalid ? 'text-gray-400' : ''
                        } ${
                            // Focus ring, visible while the cell has keyboard focus.
                            isFocusVisible
                                ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                : ''
                        } ${
                            // Darker selection background for the start and end.
                            isInvalid
                                ? 'bg-red-600 text-white hover:bg-red-700'
                                : 'bg-openg-600 text-white hover:bg-openg-700'
                        } ${
                            // Hover state for cells in the middle of the range.
                            ''
                        } ${
                            // Hover state for non-selected cells.
                            !isSelected && !isDisabled
                                ? 'hover:bg-openg-100'
                                : ''
                        } cursor-default`}
                    >
                        {formattedDate}
                    </div>
                </div>
            </td>
        )
    }
    if (isSelected && !isDisabled && !(isSelectionStart || isSelectionEnd)) {
        return (
            <td
                {...cellProps}
                className={`py-0.5 relative ${isFocusVisible ? 'z-10' : 'z-0'}`}
            >
                <div
                    {...mergeProps(buttonProps, focusProps)}
                    ref={ref}
                    hidden={isOutsideVisibleRange}
                    className={`w-10 h-10 outline-none group ${
                        isRoundedLeft ? 'rounded-l-md' : ''
                    } ${isRoundedRight ? 'rounded-r-md' : ''} ${''} ${
                        isDisabled ? 'disabled' : ''
                    }`}
                >
                    <div
                        className={`w-full h-full rounded-md flex items-center justify-center ${
                            isDisabled && !isInvalid ? 'text-gray-400' : ''
                        } ${
                            // Focus ring, visible while the cell has keyboard focus.
                            isFocusVisible
                                ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                                : ''
                        } ${
                            // Darker selection background for the start and end.
                            ''
                        } ${
                            // Hover state for cells in the middle of the range.
                            isInvalid
                                ? 'hover:bg-red-400'
                                : 'hover:bg-openg-400'
                        } ${
                            // Hover state for non-selected cells.
                            !isSelected && !isDisabled
                                ? 'hover:bg-openg-100'
                                : ''
                        } cursor-default`}
                    >
                        {formattedDate}
                    </div>
                </div>
            </td>
        )
    }
    return (
        <td
            {...cellProps}
            className={`py-0.5 relative ${isFocusVisible ? 'z-10' : 'z-0'}`}
        >
            <div
                {...mergeProps(buttonProps, focusProps)}
                ref={ref}
                hidden={isOutsideVisibleRange}
                className={`w-10 h-10 outline-none group ${
                    isRoundedLeft ? 'rounded-l-md' : ''
                } ${isRoundedRight ? 'rounded-r-md' : ''} ${''} ${
                    isDisabled ? 'disabled' : ''
                }`}
            >
                <div
                    className={`w-full h-full rounded-md flex items-center justify-center ${
                        isDisabled && !isInvalid ? 'text-gray-400' : ''
                    } ${
                        // Focus ring, visible while the cell has keyboard focus.
                        isFocusVisible
                            ? 'ring-2 group-focus:z-2 ring-openg-600 ring-offset-2'
                            : ''
                    } ${
                        // Darker selection background for the start and end.
                        ''
                    } ${
                        // Hover state for cells in the middle of the range.
                        ''
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

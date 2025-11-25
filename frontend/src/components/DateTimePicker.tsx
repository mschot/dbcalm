import { useRef, useEffect } from 'react';
import AirDatepicker from 'air-datepicker';
import localeEn from 'air-datepicker/locale/en';
import 'air-datepicker/air-datepicker.css';

interface DateTimePickerProps {
  value?: Date;
  onChange: (date: Date | null) => void;
  className?: string;
  placeholder?: string;
}

export const DateTimePicker = ({ value, onChange, className, placeholder }: DateTimePickerProps) => {
  const inputRef = useRef<HTMLInputElement>(null);
  const datepickerRef = useRef<AirDatepicker<HTMLInputElement> | null>(null);
  const onChangeRef = useRef(onChange);
  const isInternalUpdateRef = useRef(false);

  // Keep onChange ref up to date
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  // Initialize air-datepicker only once
  useEffect(() => {
    if (!inputRef.current) return;

    // Initialize air-datepicker
    datepickerRef.current = new AirDatepicker(inputRef.current, {
      locale: localeEn,
      timepicker: true,
      timeFormat: 'HH:mm',
      dateFormat: 'yyyy-MM-dd',
      position: 'bottom left',
      onSelect: ({ date }) => {
        // Only trigger onChange if this is a user selection, not a programmatic update
        if (!isInternalUpdateRef.current) {
          onChangeRef.current(date as Date | null);
        }
      },
    });

    // Cleanup on unmount
    return () => {
      if (datepickerRef.current) {
        datepickerRef.current.destroy();
      }
    };
  }, []); // Empty dependency array - initialize only once

  // Update datepicker when value prop changes
  useEffect(() => {
    if (datepickerRef.current && value !== undefined) {
      // Set flag to prevent triggering onChange callback
      isInternalUpdateRef.current = true;
      datepickerRef.current.selectDate(value);
      // Reset flag after a short delay to allow the update to complete
      setTimeout(() => {
        isInternalUpdateRef.current = false;
      }, 0);
    }
  }, [value]);

  return (
    <input
      ref={inputRef}
      type="text"
      className={className}
      placeholder={placeholder}
      readOnly
    />
  );
};

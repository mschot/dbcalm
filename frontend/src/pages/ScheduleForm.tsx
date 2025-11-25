import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../utils/api';
import { Header } from '../components/Header';
import { Schedule } from '../types/schedule';

const ScheduleForm = () => {
  const { id } = useParams<{ id: string }>();
  const isEditMode = !!id;
  const [backupType, setBackupType] = useState<'full' | 'incremental'>('full');
  const [scheduleMode, setScheduleMode] = useState<'time' | 'interval'>('time');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly' | 'hourly'>('daily');
  const [dayOfWeek, setDayOfWeek] = useState(1); // Monday
  const [dayOfMonth, setDayOfMonth] = useState(1);
  const [hour, setHour] = useState(2);
  const [minute, setMinute] = useState(0);
  const [intervalValue, setIntervalValue] = useState(15);
  const [intervalUnit, setIntervalUnit] = useState<'minutes' | 'hours'>('minutes');
  const [retentionMode, setRetentionMode] = useState<'delete' | 'forever'>('delete');
  const [retentionValue, setRetentionValue] = useState(2);
  const [retentionUnit, setRetentionUnit] = useState<'days' | 'weeks' | 'months'>('weeks');
  const [enabled, setEnabled] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (isEditMode) {
      const fetchSchedule = async () => {
        setIsLoading(true);
        try {
          const schedule = await Api.get(`/schedules/${id}`) as Schedule;
          setBackupType(schedule.backup_type);

          if (schedule.frequency === 'interval') {
            setScheduleMode('interval');
            setIntervalValue(schedule.interval_value || 15);
            setIntervalUnit(schedule.interval_unit || 'minutes');
          } else {
            setScheduleMode('time');
            setFrequency(schedule.frequency);
            setDayOfWeek(schedule.day_of_week || 1);
            setDayOfMonth(schedule.day_of_month || 1);
            setHour(schedule.hour || 2);
            setMinute(schedule.minute || 0);
          }

          if (schedule.retention_value && schedule.retention_unit) {
            setRetentionMode('delete');
            setRetentionValue(schedule.retention_value);
            setRetentionUnit(schedule.retention_unit);
          } else {
            setRetentionMode('forever');
          }

          setEnabled(schedule.enabled);
        } catch (err) {
          console.error('Failed to fetch schedule:', err);
          setError('Failed to load schedule');
        } finally {
          setIsLoading(false);
        }
      };

      fetchSchedule();
    }
  }, [id, isEditMode]);

  const getOrdinalSuffix = (day: number): string => {
    if (day >= 11 && day <= 13) return 'th';
    switch (day % 10) {
      case 1: return 'st';
      case 2: return 'nd';
      case 3: return 'rd';
      default: return 'th';
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setIsSubmitting(true);
    setError(null);

    // Validate interval values
    if (scheduleMode === 'interval') {
      if (intervalUnit === 'minutes' && intervalValue > 59) {
        setError('Interval value for minutes cannot exceed 59');
        setIsSubmitting(false);
        return;
      }
      if (intervalUnit === 'hours' && intervalValue > 23) {
        setError('Interval value for hours cannot exceed 23');
        setIsSubmitting(false);
        return;
      }
      if (intervalValue < 1) {
        setError('Interval value must be at least 1');
        setIsSubmitting(false);
        return;
      }
    }

    // Validate retention values
    if (retentionMode === 'delete' && retentionValue < 1) {
      setError('Retention value must be at least 1');
      setIsSubmitting(false);
      return;
    }

    try {
      const scheduleData: {
        backup_type: string;
        frequency: string;
        day_of_week?: number;
        day_of_month?: number;
        hour?: number;
        minute?: number;
        interval_value?: number;
        interval_unit?: string;
        retention_value?: number;
        retention_unit?: string;
        enabled: boolean;
      } = {
        backup_type: backupType,
        frequency: scheduleMode === 'interval' ? 'interval' : frequency,
        enabled,
      };

      if (scheduleMode === 'interval') {
        scheduleData.interval_value = intervalValue;
        scheduleData.interval_unit = intervalUnit;
      } else {
        scheduleData.hour = hour;
        scheduleData.minute = minute;

        // Add day_of_week for weekly schedules
        if (frequency === 'weekly') {
          scheduleData.day_of_week = dayOfWeek;
        }

        // Add day_of_month for monthly schedules
        if (frequency === 'monthly') {
          scheduleData.day_of_month = dayOfMonth;
        }
      }

      // Add retention settings if delete mode is selected
      if (retentionMode === 'delete') {
        scheduleData.retention_value = retentionValue;
        scheduleData.retention_unit = retentionUnit;
      }

      if (isEditMode) {
        await Api.put(`/schedules/${id}`, scheduleData);
      } else {
        await Api.post('/schedules', scheduleData);
      }

      navigate('/schedules');
    } catch (err) {
      console.error(`Failed to ${isEditMode ? 'update' : 'create'} schedule:`, err);
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(`Failed to ${isEditMode ? 'update' : 'create'} schedule: ${message}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Generate hour options (00-23)
  const hourOptions = Array.from({ length: 24 }, (_, i) => ({
    value: i,
    label: i.toString().padStart(2, '0'),
  }));

  // Generate minute options (00-59)
  const minuteOptions = Array.from({ length: 60 }, (_, i) => ({
    value: i,
    label: i.toString().padStart(2, '0'),
  }));

  // Day of week options
  const dayOfWeekOptions = [
    { value: 1, label: 'Monday' },
    { value: 2, label: 'Tuesday' },
    { value: 3, label: 'Wednesday' },
    { value: 4, label: 'Thursday' },
    { value: 5, label: 'Friday' },
    { value: 6, label: 'Saturday' },
    { value: 0, label: 'Sunday' },
  ];

  // Day of month options (1-28)
  const dayOfMonthOptions = Array.from({ length: 28 }, (_, i) => ({
    value: i + 1,
    label: (i + 1).toString(),
  }));

  if (isLoading) {
    return (
      <div className="min-h-screen bg-base-200 p-4">
        <div className="container mx-auto">
          <Header currentPage="schedules" />
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <div className="flex justify-center items-center py-12">
                <span className="loading loading-spinner loading-lg"></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="schedules" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body">
            <h2 className="card-title text-2xl mb-6">
              {isEditMode ? 'Edit Schedule' : 'Add New Schedule'}
            </h2>

            {error && (
              <div className="alert alert-error mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{error}</span>
              </div>
            )}

            <form onSubmit={handleSubmit}>

               {/* Backup Type */}
              <div className="form-control mb-4">
                <label className="label">
                  <span className="label-text">Backup Type</span>
                </label>
                <div className="flex gap-4 mt-2">
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="backup_type"
                      className="radio"
                      checked={backupType === 'full'}
                      onChange={() => setBackupType('full')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">Full Backup</span>
                  </label>
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="backup_type"
                      className="radio"
                      checked={backupType === 'incremental'}
                      onChange={() => setBackupType('incremental')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">Incremental Backup</span>
                  </label>
                </div>
              </div>

              {/* Schedule Mode Selection */}
              <h3 className="text-lg font-semibold mb-2">Schedule:</h3>
              <div className="form-control mb-4">
                <div className="flex gap-4 mt-2">
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="schedule_mode"
                      className="radio"
                      checked={scheduleMode === 'time'}
                      onChange={() => setScheduleMode('time')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">At specific time</span>
                  </label>
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="schedule_mode"
                      className="radio"
                      checked={scheduleMode === 'interval'}
                      onChange={() => setScheduleMode('interval')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">Every X minutes/hours</span>
                  </label>
                </div>
              </div>

              {/* Time-based Schedule */}
              {scheduleMode === 'time' && (
                <div className="form-control mb-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="label-text">Run once {frequency === 'hourly' ? 'an' : 'a'}</span>
                    <select
                      className="select select-bordered w-auto"
                      value={frequency}
                      onChange={(e) => setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly' | 'hourly')}
                      disabled={isSubmitting}
                    >
                      <option value="hourly">hour</option>
                      <option value="daily">day</option>
                      <option value="weekly">week</option>
                      <option value="monthly">month</option>
                    </select>

                    {frequency === 'weekly' && (
                      <>
                        <span className="label-text">on a</span>
                        <select
                          className="select select-bordered w-auto"
                          value={dayOfWeek}
                          onChange={(e) => setDayOfWeek(Number(e.target.value))}
                          disabled={isSubmitting}
                        >
                          {dayOfWeekOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </select>
                      </>
                    )}

                    {frequency === 'monthly' && (
                      <>
                        <span className="label-text">on the</span>
                        <select
                          className="select select-bordered w-auto"
                          value={dayOfMonth}
                          onChange={(e) => setDayOfMonth(Number(e.target.value))}
                          disabled={isSubmitting}
                        >
                          {dayOfMonthOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </select>
                        <span className="label-text">{getOrdinalSuffix(dayOfMonth)} day of the month</span>
                      </>
                    )}

                    {frequency === 'hourly' && (
                      <>
                        <span className="label-text">at the</span>
                        <select
                          className="select select-bordered w-auto"
                          value={minute}
                          onChange={(e) => setMinute(Number(e.target.value))}
                          disabled={isSubmitting}
                        >
                          {minuteOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </select>
                        <span className="label-text">{getOrdinalSuffix(minute)} minute</span>
                      </>
                    )}

                    {(frequency === 'daily' || frequency === 'weekly' || frequency === 'monthly') && (
                      <>
                        <span className="label-text">at</span>
                        <div className="flex items-center gap-1">
                          <select
                            className="select select-bordered w-auto"
                            value={hour}
                            onChange={(e) => setHour(Number(e.target.value))}
                            disabled={isSubmitting}
                          >
                            {hourOptions.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </select>
                          <span className="label-text">:</span>
                          <select
                            className="select select-bordered w-auto"
                            value={minute}
                            onChange={(e) => setMinute(Number(e.target.value))}
                            disabled={isSubmitting}
                          >
                            {minuteOptions.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </select>
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )}

              {/* Interval-based Schedule */}
              {scheduleMode === 'interval' && (
                <div className="form-control mb-4">
                  <label className="label">
                    <span className="label-text">Run every</span>
                  </label>
                  <div className="flex gap-2 items-center">
                    <input
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      className="input input-bordered w-24"
                      value={intervalValue}
                      onChange={(e) => {
                        const value = e.target.value.replace(/[^0-9]/g, '');
                        if (value === '') {
                          setIntervalValue(0);
                        } else {
                          setIntervalValue(Number(value));
                        }
                      }}
                      onBlur={(e) => {
                        if (intervalValue === 0 || e.target.value === '') {
                          setIntervalValue(1);
                        }
                      }}
                      disabled={isSubmitting}
                    />
                    <select
                      className="select select-bordered w-auto"
                      value={intervalUnit}
                      onChange={(e) => setIntervalUnit(e.target.value as 'minutes' | 'hours')}
                      disabled={isSubmitting}
                    >
                      <option value="minutes">Minutes</option>
                      <option value="hours">Hours</option>
                    </select>
                  </div>
                </div>
              )}

              {/* Retention Settings */}
              <h3 className="text-lg font-semibold mb-2 mt-6">Retention:</h3>
              <div className="form-control mb-4">
                <div className="flex gap-4 mt-2">
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="retention_mode"
                      className="radio"
                      checked={retentionMode === 'delete'}
                      onChange={() => setRetentionMode('delete')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">Delete backups after</span>
                  </label>
                  <label className="label cursor-pointer gap-2">
                    <input
                      type="radio"
                      name="retention_mode"
                      className="radio"
                      checked={retentionMode === 'forever'}
                      onChange={() => setRetentionMode('forever')}
                      disabled={isSubmitting}
                    />
                    <span className="label-text">Keep backups forever</span>
                  </label>
                </div>

                {retentionMode === 'delete' && (
                  <div className="flex gap-2 items-center mt-2">
                    <input
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      className="input input-bordered w-24"
                      value={retentionValue}
                      onChange={(e) => {
                        const value = e.target.value.replace(/[^0-9]/g, '');
                        if (value === '') {
                          setRetentionValue(0);
                        } else {
                          setRetentionValue(Number(value));
                        }
                      }}
                      onBlur={(e) => {
                        if (retentionValue === 0 || e.target.value === '') {
                          setRetentionValue(1);
                        }
                      }}
                      disabled={isSubmitting}
                    />
                    <select
                      className="select select-bordered w-auto"
                      value={retentionUnit}
                      onChange={(e) => setRetentionUnit(e.target.value as 'days' | 'weeks' | 'months')}
                      disabled={isSubmitting}
                    >
                      <option value="days">Days</option>
                      <option value="weeks">Weeks</option>
                      <option value="months">Months</option>
                    </select>
                  </div>
                )}
              </div>

              {/* Enabled status - only show in edit mode */}
              {isEditMode && (
                <div className="form-control mb-4">
                  <label className="label cursor-pointer justify-start gap-4">
                    <input
                      type="checkbox"
                      className="toggle toggle-success"
                      checked={enabled}
                      onChange={(e) => setEnabled(e.target.checked)}
                      disabled={isSubmitting}
                    />
                    <span className="label-text font-semibold">
                      {enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </label>
                </div>
              )}

              {/* Form Actions */}
              <div className="form-control mt-6 flex flex-row gap-2 justify-end">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => navigate('/schedules')}
                  disabled={isSubmitting}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className={`btn btn-primary ${isSubmitting ? 'loading' : ''}`}
                  disabled={isSubmitting}
                >
                  {isSubmitting ? (isEditMode ? 'Updating...' : 'Creating...') : (isEditMode ? 'Update Schedule' : 'Create Schedule')}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ScheduleForm;

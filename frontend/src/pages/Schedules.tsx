import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Api } from '../utils/api';
import { Schedule } from '../types/schedule';
import { ScheduleResponse } from '../types/scheduleResponse';
import { Header } from '../components/Header';
import { Pagination, PaginationResponse } from '../components/Pagination';
import { BackupTypeIcon } from '../components/BackupTypeIcon';

const Schedules = () => {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationResponse, setPaginationResponse] = useState<PaginationResponse>({
    total: 0,
    page: 1,
    per_page: 25,
    total_pages: 1
  });
  const [openMenuId, setOpenMenuId] = useState<number | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchSchedules = async () => {
      try {
        const response = await Api.get(`/schedules?page=${currentPage}`) as ScheduleResponse;
        setSchedules(response.items);
        setPaginationResponse(response.pagination);
      } catch (error) {
        console.error('Failed to fetch schedules:', error);
      }
    };

    fetchSchedules();
  }, [currentPage]);

  const handleDelete = async (id: number) => {
    const isConfirmed = window.confirm(
      "Are you sure you want to delete this schedule? The automated backups will stop."
    );

    if (isConfirmed) {
      try {
        await Api.delete(`/schedules/${id}`);
        setSchedules(schedules.filter(schedule => schedule.id !== id));
      } catch (error) {
        console.error('Failed to delete schedule:', error);
        alert('Failed to delete the schedule. Please try again.');
      }
    }
  };

  const handleToggleEnabled = async (schedule: Schedule) => {
    try {
      const updatedSchedule = { ...schedule, enabled: !schedule.enabled };
      await Api.put(`/schedules/${schedule.id}`, updatedSchedule);

      setSchedules(schedules.map(s =>
        s.id === schedule.id ? { ...s, enabled: !s.enabled } : s
      ));
    } catch (error) {
      console.error('Failed to toggle schedule:', error);
      alert('Failed to update the schedule. Please try again.');
    }
  };

  const formatScheduleTime = (schedule: Schedule): string => {
    if (schedule.frequency === 'interval') {
      return `Every ${schedule.interval_value} ${schedule.interval_unit}`;
    }

    const minuteStr = (schedule.minute || 0).toString().padStart(2, '0');

    if (schedule.frequency === 'hourly') {
      const suffix = (schedule.minute || 0) >= 11 && (schedule.minute || 0) <= 13 ? 'th' :
                     ((schedule.minute || 0) % 10 === 1 ? 'st' :
                      (schedule.minute || 0) % 10 === 2 ? 'nd' :
                      (schedule.minute || 0) % 10 === 3 ? 'rd' : 'th');
      return `Hourly at the ${schedule.minute}${suffix} minute`;
    }

    const hourStr = (schedule.hour || 0).toString().padStart(2, '0');
    const time = `${hourStr}:${minuteStr}`;

    if (schedule.frequency === 'daily') {
      return `Daily at ${time}`;
    } else if (schedule.frequency === 'weekly') {
      const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
      const dayName = days[schedule.day_of_week || 0];
      return `${dayName}s at ${time}`;
    } else if (schedule.frequency === 'monthly') {
      const day = schedule.day_of_month || 1;
      const suffix = day === 1 ? 'st' : day === 2 ? 'nd' : day === 3 ? 'rd' : 'th';
      return `${day}${suffix} of month at ${time}`;
    }
    return time;
  };

  const formatRetention = (schedule: Schedule): string => {
    if (schedule.retention_value && schedule.retention_unit) {
      return `${schedule.retention_value} ${schedule.retention_unit}`;
    }
    return 'Forever';
  };

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="schedules" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body p-0">
            <div>
              <table className="table table-zebra w-full">
                <thead>
                  <tr>
                    <th className="text-base-content">Type</th>
                    <th className="text-base-content">Schedule</th>
                    <th className="text-base-content">Retention</th>
                    <th className="text-base-content">Status</th>
                    <th className="text-right text-base-content"></th>
                  </tr>
                </thead>
                <tbody>
                  {schedules.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="text-center py-8 text-gray-500">
                        No schedules found. Click the + button to create one.
                      </td>
                    </tr>
                  ) : (
                    schedules.map((schedule) => (
                      <tr key={schedule.id} className="hover">
                        <td>
                          <BackupTypeIcon type={schedule.backup_type} />
                        </td>
                        <td>{formatScheduleTime(schedule)}</td>
                        <td>{formatRetention(schedule)}</td>
                        <td>
                          <button
                            onClick={() => handleToggleEnabled(schedule)}
                            className={`badge cursor-pointer ${
                              schedule.enabled ? 'badge-success' : 'badge-error'
                            }`}
                          >
                            {schedule.enabled ? 'Enabled' : 'Disabled'}
                          </button>
                        </td>
                        <td className="text-right">
                          <div className="dropdown dropdown-end">
                            <button
                              onClick={() => setOpenMenuId(openMenuId === schedule.id ? null : schedule.id)}
                              className="btn btn-ghost btn-sm btn-circle"
                            >
                              <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                              </svg>
                            </button>
                            {openMenuId === schedule.id && (
                              <ul className="dropdown-content menu menu-sm bg-base-200 rounded-box w-52 p-2 shadow-lg">
                                <li>
                                  <button onClick={() => {
                                    navigate(`/schedules/edit/${schedule.id}`);
                                    setOpenMenuId(null);
                                  }}>
                                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                    </svg>
                                    Edit
                                  </button>
                                </li>
                                <li>
                                  <button
                                    onClick={() => {
                                      handleDelete(schedule.id);
                                      setOpenMenuId(null);
                                    }}
                                    className="text-error"
                                  >
                                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                    Delete
                                  </button>
                                </li>
                              </ul>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <Pagination
              paginationResponse={paginationResponse}
              onPageChange={setCurrentPage}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Schedules;

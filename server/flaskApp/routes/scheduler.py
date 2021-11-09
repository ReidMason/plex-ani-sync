from typing import Optional

from apscheduler.job import Job
from config import Config
from flask import Blueprint, jsonify, request
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from threading import Thread
from syncRunner import SyncRunner
import utils
from ..app import socketio

logger = utils.create_logger("Scheduler")

scheduler_route = Blueprint('scheduler', __name__, url_prefix = '/api/scheduler')

job_running = False


def emit_sync_process_logs(payload: dict):
    socketio.emit('sync_process_logs', payload)


def run_job_threading():
    global job_running
    job_running = True
    logger.debug("Running syncher...")
    SyncRunner(emit_sync_process_logs).run()
    logger.debug("Syncher finished")
    job_running = False


def run_job():
    if job_running:
        print("Job already running")
        return

    # socketio.start_background_task(run_job_threading)
    # socketio.start_background_task(target = run_job_threading)
    thread = Thread(target = run_job_threading)
    thread.start()


def set_scheduler_job(config: Config = None) -> Optional[Job]:
    config = config if config is not None else Config()
    # Catch invalid cron format
    try:
        cron_trigger = CronTrigger.from_crontab(config.SYNC_CRONTIME)
    except Exception as e:
        logger.error(f"Failed to update sync time. {e}")
        return None

    scheduler.remove_all_jobs()
    job = scheduler.add_job(run_job, cron_trigger)
    next_run_time = job.next_run_time.strftime(config.DATE_FORMAT)
    logger.info(f"Sync time updated next run at: {next_run_time}")
    return job


# Start the scheduler
scheduler = BackgroundScheduler()
scheduler.start()
set_scheduler_job()


@scheduler_route.route('/forceRunSync', methods = ["POST"])
def force_run_sync():
    if job_running:
        return jsonify({"message": "Job already running"})

    run_job()
    return {"message": "Job started"}


@scheduler_route.route('/getNextRunTime')
def get_next_run_time():
    jobs = scheduler.get_jobs()
    if len(jobs) == 0:
        return jsonify({})

    config = Config()
    job = jobs[0]
    next_run_time = job.next_run_time.strftime(config.DATE_FORMAT)
    return jsonify({"nextRunTime": next_run_time, "syncRunning": job_running})


@scheduler_route.route('/updateScheduler', methods = ["POST"])
def update_scheduler():
    config = Config()
    data = request.json

    # Set scheduler enabled
    sync_schedule_enabled = data.get('syncScheduleEnabled')
    if sync_schedule_enabled is not None:
        config.SYNC_SCHEDULE_ENABLED = sync_schedule_enabled
        config.save()

    new_crontime = data.get('crontime')

    # Validate new crontime
    try:
        CronTrigger.from_crontab(new_crontime)
    except Exception as e:
        return jsonify({"Error": e.args[0]})

    # Update sync time
    config.SYNC_CRONTIME = new_crontime
    config.save()

    job = set_scheduler_job(config)
    next_run_time = job.next_run_time.strftime(config.DATE_FORMAT)
    return jsonify({"nextRunTime": next_run_time, "syncScheduleEnabled": config.SYNC_SCHEDULE_ENABLED})

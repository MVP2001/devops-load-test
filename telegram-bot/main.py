import os
import time
import requests
import asyncio
from telegram import Bot

TELEGRAM_TOKEN = os.getenv('TELEGRAM_BOT_TOKEN')
CHAT_ID = os.getenv('TELEGRAM_CHAT_ID')
API_URL = os.getenv('API_URL', 'http://backend:8080')

bot = Bot(token=TELEGRAM_TOKEN)

async def send_alert(message):
    await bot.send_message(chat_id=CHAT_ID, text=message, parse_mode='Markdown')

def check_metrics():
    try:
        response = requests.get(f'{API_URL}/api/v1/metrics/realtime', timeout=5)
        data = response.json()
        
        alerts = []
        
        if data['cpu']['usage_percent'] > 90:
            alerts.append(f"🚨 *CPU Critical*: {data['cpu']['usage_percent']:.1f}%")
        
        if data['memory']['usage_percent'] > 90:
            alerts.append(f"🚨 *Memory Critical*: {data['memory']['usage_percent']:.1f}%")
        
        if data['disk']['usage_percent'] > 95:
            alerts.append(f"🚨 *Disk Critical*: {data['disk']['usage_percent']:.1f}%")
        
        return alerts
    except Exception as e:
        return [f"❌ *Error checking metrics*: {str(e)}"]

async def main():
    await send_alert("✅ *DevOps Load Platform Monitor Started*\nTarget: 185.40.76.46")
    
    while True:
        alerts = check_metrics()
        for alert in alerts:
            await send_alert(alert)
        await asyncio.sleep(30)

if __name__ == '__main__':
    asyncio.run(main())
